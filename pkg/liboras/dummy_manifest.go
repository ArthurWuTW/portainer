package liboras

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/encoding/json"
	"oras.land/oras-go/v2/registry/remote"
)

// generateMinimalManifest creates a minimal OCI manifest with empty config and no layers
func generateMinimalManifest() (*ocispec.Manifest, []byte, error) {
	// Create empty config blob
	emptyConfig := []byte("{}")
	configDescriptor := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.config.v1+json",
		Digest:    digest.FromBytes(emptyConfig), // sha256 of empty JSON object "{}"
		Size:      int64(len(emptyConfig)),
	}

	// Create minimal manifest with no layers
	manifest := &ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Config:    configDescriptor,
		Layers:    []ocispec.Descriptor{}, // Empty layers array
	}

	// Marshal manifest to JSON
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal dummy manifest: %w", err)
	}

	return manifest, manifestBytes, nil
}

// CreateDummyManifest creates a minimal dummy manifest in the repository and returns its digest
func CreateDummyManifest(registryClient *remote.Registry, repository string) (string, error) {
	ctx := context.Background()

	// Get repository handle
	repo, err := registryClient.Repository(ctx, repository)
	if err != nil {
		return "", fmt.Errorf("failed to get repository handle: %w", err)
	}

	// Generate minimal manifest
	manifest, manifestBytes, err := generateMinimalManifest()
	if err != nil {
		return "", fmt.Errorf("failed to generate minimal manifest: %w", err)
	}

	// First, push the empty config blob
	emptyConfig := []byte("{}")
	configReader := bytes.NewReader(emptyConfig)
	configDescriptor := ocispec.Descriptor{
		MediaType: "application/vnd.oci.image.config.v1+json",
		Digest:    digest.FromBytes(emptyConfig),
		Size:      int64(len(emptyConfig)),
	}

	err = repo.Blobs().Push(ctx, configDescriptor, configReader)
	if err != nil {
		return "", fmt.Errorf("failed to push config blob: %w", err)
	}

	// Then push the manifest with a temporary tag
	manifestReader := bytes.NewReader(manifestBytes)
	manifestDescriptor := ocispec.Descriptor{
		MediaType: manifest.MediaType,
		Size:      int64(len(manifestBytes)),
		Digest:    digest.FromBytes(manifestBytes),
	}

	// Use a unique temporary tag name for the dummy manifest
	dummyTag := fmt.Sprintf("__portainer_dummy_%d", time.Now().UnixNano())

	err = repo.Manifests().PushReference(ctx, manifestDescriptor, manifestReader, dummyTag)
	if err != nil {
		return "", fmt.Errorf("failed to push dummy manifest: %w", err)
	}

	// Return the manifest digest directly from what we calculated
	return manifestDescriptor.Digest.String(), nil
}

// PointTagToDummy updates a tag to point to the dummy manifest
func PointTagToDummy(registryClient *remote.Registry, repository, tagName, dummyDigest string) error {
	// Generate the same minimal manifest content
	_, manifestBytes, err := generateMinimalManifest()
	if err != nil {
		return fmt.Errorf("failed to generate minimal manifest: %w", err)
	}

	return AddTagToManifest(registryClient, repository, tagName, dummyDigest, manifestBytes)
}

// DeleteTag deletes a single tag without affecting sibling tags pointing to
// the same manifest. When no other tag points at the manifest anymore, the
// manifest itself is deleted as well, so no untagged image is left behind and
// the repository disappears once its last tag is removed.
func DeleteTag(registryClient *remote.Registry, repository, tag string) error {
	ctx := context.Background()

	repo, err := registryClient.Repository(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to get repository handle: %w", err)
	}

	descriptor, err := repo.Resolve(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to resolve tag %q: %w", tag, err)
	}

	tags, err := ListTags(ctx, registryClient, repository)
	if err != nil {
		return fmt.Errorf("failed to list tags of repository %q: %w", repository, err)
	}

	for _, other := range tags {
		if other == tag {
			continue
		}

		otherDescriptor, err := repo.Resolve(ctx, other)
		if err != nil {
			// The sibling manifest is unreachable, so it is unknown whether it
			// shares the manifest of the tag being deleted. Fall back to the
			// dummy manifest flow, which can only ever remove the target tag.
			return SafeDeleteTags(registryClient, repository, []string{tag})
		}

		if otherDescriptor.Digest == descriptor.Digest {
			// Another tag shares the manifest: only untag the requested tag.
			return SafeDeleteTags(registryClient, repository, []string{tag})
		}
	}

	// The deleted tag is the only reference to the manifest, so removing the
	// manifest removes the tag and leaves no untagged image behind.
	return DeleteManifestByDigest(registryClient, repository, descriptor.Digest.String())
}

// SafeDeleteTags safely deletes multiple tags without affecting others pointing to the same manifest
func SafeDeleteTags(registryClient *remote.Registry, repository string, tagsToDelete []string) error {
	if len(tagsToDelete) == 0 {
		return nil
	}

	// Create a dummy manifest for all tags to delete
	dummyDigest, err := CreateDummyManifest(registryClient, repository)
	if err != nil {
		return fmt.Errorf("failed to create dummy manifest: %w", err)
	}

	// Point all tags to the same dummy manifest
	for _, tagToDelete := range tagsToDelete {
		err = PointTagToDummy(registryClient, repository, tagToDelete, dummyDigest)
		if err != nil {
			// Cleanup: delete dummy manifest on failure
			cleanupErr := DeleteManifestByDigest(registryClient, repository, dummyDigest)
			if cleanupErr != nil {
				log.Warn().
					Err(cleanupErr).
					Str("repository", repository).
					Str("tag", tagToDelete).
					Str("digest", dummyDigest).
					Msg("Failed to cleanup dummy manifest after tag pointing error")
				return fmt.Errorf("failed to point tag %s to dummy: %w (cleanup also failed: %w)", tagToDelete, err, cleanupErr)
			}
			return fmt.Errorf("failed to point tag %s to dummy: %w", tagToDelete, err)
		}
	}

	// Delete the dummy manifest (removes ALL pointed tags safely)
	err = DeleteManifestByDigest(registryClient, repository, dummyDigest)
	if err != nil {
		log.Error().
			Err(err).
			Str("repository", repository).
			Str("digest", dummyDigest).
			Int("tag_count", len(tagsToDelete)).
			Msg("Failed to delete dummy manifest containing tags")
		return fmt.Errorf("failed to delete dummy manifest: %w", err)
	}

	return nil
}

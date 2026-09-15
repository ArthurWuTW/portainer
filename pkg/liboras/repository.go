package liboras

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	portainer "github.com/portainer/portainer/api"
	"github.com/portainer/portainer/api/concurrent"
	"github.com/portainer/portainer/api/logs"
	"github.com/segmentio/encoding/json"
	"golang.org/x/mod/semver"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
)

// ListRepositories retrieves all repositories from a registry using specialized repository listing clients
// Each registry type has different repository listing implementations that require specific API calls
func ListRepositories(ctx context.Context, registry *portainer.Registry, registryClient *remote.Registry) ([]string, error) {
	factory := NewRepositoryListClientFactory()
	listClient, err := factory.CreateListClientWithRegistry(registry, registryClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository list client: %w", err)
	}

	return listClient.ListRepositories(ctx)
}

// ListTags retrieves all tags of a repository
func ListTags(ctx context.Context, registryClient *remote.Registry, repositoryName string) ([]string, error) {
	repository, err := registryClient.Repository(ctx, repositoryName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository handle: %w", err)
	}

	var tags []string
	err = repository.Tags(ctx, "", func(tagList []string) error {
		tags = append(tags, tagList...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags of repository %q: %w", repositoryName, err)
	}

	return tags, nil
}

// TagInfo holds a tag together with the metadata read from its manifest
type TagInfo struct {
	Tag     string
	Digest  string
	Size    int64
	Created time.Time
}

// manifestEnvelope covers an image manifest, a manifest list and an OCI index,
// as all of them are JSON documents sharing most of their fields
type manifestEnvelope struct {
	ocispec.Manifest
	Manifests []ocispec.Descriptor `json:"manifests,omitempty"`
}

// maxManifestDepth limits how deep manifest lists are followed looking for a creation date
const maxManifestDepth = 3

// tagInspectConcurrency is the number of tags inspected concurrently
const tagInspectConcurrency = 10

// ListTagsWithInfo retrieves all tags of a repository enriched with the manifest digest,
// size and image creation date. Metadata is read on a best effort basis, so a tag pointing
// to an unreachable manifest is still returned, without any metadata.
func ListTagsWithInfo(ctx context.Context, registryClient *remote.Registry, repositoryName string) ([]TagInfo, error) {
	repository, err := registryClient.Repository(ctx, repositoryName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository handle: %w", err)
	}

	var tags []string
	err = repository.Tags(ctx, "", func(tagList []string) error {
		tags = append(tags, tagList...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags of repository %q: %w", repositoryName, err)
	}

	var tasks []concurrent.Func
	for _, tag := range tags {
		task := func(ctx context.Context) (any, error) {
			return describeTag(ctx, repository, tag), nil
		}
		tasks = append(tasks, task)
	}

	results, err := concurrent.Run(ctx, tagInspectConcurrency, tasks...)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect tags of repository %q: %w", repositoryName, err)
	}

	tagInfos := make([]TagInfo, 0, len(results))
	for _, result := range results {
		if tagInfo, ok := result.Result.(TagInfo); ok {
			tagInfos = append(tagInfos, tagInfo)
		}
	}

	sort.Slice(tagInfos, func(i, j int) bool {
		return tagInfos[i].Tag < tagInfos[j].Tag
	})

	return tagInfos, nil
}

// describeTag resolves a tag and collects the metadata of the manifest it points to
func describeTag(ctx context.Context, repository registry.Repository, tag string) TagInfo {
	tagInfo := TagInfo{Tag: tag}

	descriptor, err := repository.Resolve(ctx, tag)
	if err != nil {
		return tagInfo
	}

	tagInfo.Digest = descriptor.Digest.String()
	tagInfo.Size = descriptor.Size
	tagInfo.Created = findCreatedDate(ctx, repository, descriptor, 0)

	return tagInfo
}

// findCreatedDate looks for the creation date in the manifest annotations, then in the
// referenced image config, then in the first child manifest of a manifest list.
// It returns a zero time when the registry exposes no creation date.
func findCreatedDate(ctx context.Context, repository registry.Repository, descriptor ocispec.Descriptor, depth int) time.Time {
	if depth >= maxManifestDepth {
		return time.Time{}
	}

	manifest, err := fetchManifest(ctx, repository, descriptor)
	if err != nil {
		return time.Time{}
	}

	if created, ok := parseCreatedDate(manifest.Annotations[ocispec.AnnotationCreated]); ok {
		return created
	}

	if created := fetchConfigCreatedDate(ctx, repository, manifest.Config); !created.IsZero() {
		return created
	}

	if len(manifest.Manifests) > 0 {
		return findCreatedDate(ctx, repository, manifest.Manifests[0], depth+1)
	}

	return time.Time{}
}

// fetchManifest reads and decodes the manifest content of a descriptor
func fetchManifest(ctx context.Context, repository registry.Repository, descriptor ocispec.Descriptor) (manifestEnvelope, error) {
	var manifest manifestEnvelope

	manifestReader, err := repository.Manifests().Fetch(ctx, descriptor)
	if err != nil {
		return manifest, err
	}
	defer logs.CloseAndLogErr(manifestReader)

	content, err := io.ReadAll(manifestReader)
	if err != nil {
		return manifest, err
	}

	if err := json.Unmarshal(content, &manifest); err != nil {
		return manifest, err
	}

	return manifest, nil
}

// fetchConfigCreatedDate reads the 'created' field out of an image config blob
func fetchConfigCreatedDate(ctx context.Context, repository registry.Repository, config ocispec.Descriptor) time.Time {
	if config.Digest == "" {
		return time.Time{}
	}

	configReader, err := repository.Blobs().Fetch(ctx, config)
	if err != nil {
		return time.Time{}
	}
	defer logs.CloseAndLogErr(configReader)

	content, err := io.ReadAll(configReader)
	if err != nil {
		return time.Time{}
	}

	var imageConfig struct {
		Created *time.Time `json:"created"`
	}
	if err := json.Unmarshal(content, &imageConfig); err != nil || imageConfig.Created == nil {
		return time.Time{}
	}

	return imageConfig.Created.UTC()
}

// parseCreatedDate parses an RFC 3339 timestamp as found in OCI annotations
func parseCreatedDate(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}

	created, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}

	return created.UTC(), true
}

// FilterRepositoriesByMediaType filters repositories to only include those with the expected media type
func FilterRepositoriesByMediaType(ctx context.Context, repositoryNames []string, registryClient *remote.Registry, expectedMediaType string) ([]string, error) {
	// Run concurrently as this can take 10s+ to complete in serial
	var tasks []concurrent.Func
	for _, repoName := range repositoryNames {
		name := repoName
		task := func(ctx context.Context) (any, error) {
			repository, err := registryClient.Repository(ctx, name)
			if err != nil {
				return nil, err
			}

			if HasMediaType(ctx, repository, expectedMediaType) {
				return name, nil
			}
			return nil, nil // not a repository with the expected media type
		}
		tasks = append(tasks, task)
	}

	// 10 is a reasonable max concurrency limit
	results, err := concurrent.Run(ctx, 10, tasks...)
	if err != nil {
		return nil, err
	}

	// Collect repository names
	var repositories []string
	for _, result := range results {
		if result.Result != nil {
			if repoName, ok := result.Result.(string); ok {
				repositories = append(repositories, repoName)
			}
		}
	}

	return repositories, nil
}

// HasMediaType checks if a repository has artifacts with the specified media type
func HasMediaType(ctx context.Context, repository registry.Repository, expectedMediaType string) bool {
	// Check the first available tag
	// Reasonable limitation - it won't work for repos where the latest tag is missing the expected media type but other tags have it
	// This tradeoff is worth it for the performance benefits
	var latestTag string
	err := repository.Tags(ctx, "", func(tagList []string) error {
		if len(tagList) > 0 {
			// Order the taglist by latest semver, then get the latest tag
			// e.g. ["1.0", "1.1"] -> ["1.1", "1.0"] -> "1.1"
			sort.Slice(tagList, func(i, j int) bool {
				return semver.Compare(tagList[i], tagList[j]) > 0
			})
			latestTag = tagList[0]
		}
		return nil
	})

	if err != nil {
		return false
	}

	if latestTag == "" {
		return false
	}

	descriptor, err := repository.Resolve(ctx, latestTag)
	if err != nil {
		return false
	}

	return descriptorHasMediaType(ctx, repository, descriptor, expectedMediaType)
}

// descriptorHasMediaType checks if a descriptor or its manifest contains the expected media type
func descriptorHasMediaType(ctx context.Context, repository registry.Repository, descriptor ocispec.Descriptor, expectedMediaType string) bool {
	// Check if the descriptor indicates the expected media type
	if descriptor.MediaType == expectedMediaType {
		return true
	}

	// Otherwise, look for the expected media type in the entire manifest content
	manifestReader, err := repository.Manifests().Fetch(ctx, descriptor)
	if err != nil {
		return false
	}
	defer logs.CloseAndLogErr(manifestReader)

	content, err := io.ReadAll(manifestReader)
	if err != nil {
		return false
	}
	var manifest ocispec.Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return false
	}

	return manifest.Config.MediaType == expectedMediaType
}

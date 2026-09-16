package liboras

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	manifestDigestA = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	manifestDigestB = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

type mockRegistry struct {
	server *httptest.Server

	tagDigests map[string]string

	deletedRefs        []string
	pushedManifestTags []string
	blobUploadStarted  bool
}

func newMockRegistry(t *testing.T, tagDigests map[string]string) *mockRegistry {
	t.Helper()

	mock := &mockRegistry{tagDigests: tagDigests}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == "/v2/test-repo/tags/list" && r.Method == http.MethodGet:
			tags := make([]string, 0, len(mock.tagDigests))
			for tag := range mock.tagDigests {
				tags = append(tags, tag)
			}
			quoted := make([]string, len(tags))
			for i, tag := range tags {
				quoted[i] = fmt.Sprintf("%q", tag)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"name":"test-repo","tags":[%s]}`, strings.Join(quoted, ","))

		case strings.HasPrefix(path, "/v2/test-repo/manifests/") && r.Method == http.MethodHead:
			reference := strings.TrimPrefix(path, "/v2/test-repo/manifests/")
			manifestDigest, ok := mock.tagDigests[reference]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Header().Set("Docker-Content-Digest", manifestDigest)
			w.Header().Set("Content-Length", "2")
			w.WriteHeader(http.StatusOK)

		case strings.HasPrefix(path, "/v2/test-repo/manifests/") && r.Method == http.MethodDelete:
			mock.deletedRefs = append(mock.deletedRefs, strings.TrimPrefix(path, "/v2/test-repo/manifests/"))
			w.WriteHeader(http.StatusAccepted)

		case strings.HasPrefix(path, "/v2/test-repo/manifests/") && r.Method == http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			mock.pushedManifestTags = append(mock.pushedManifestTags, strings.TrimPrefix(path, "/v2/test-repo/manifests/"))
			hash := sha256.Sum256(body)
			w.Header().Set("Docker-Content-Digest", fmt.Sprintf("sha256:%x", hash))
			w.WriteHeader(http.StatusCreated)

		case strings.HasPrefix(path, "/v2/test-repo/blobs/uploads/") && r.Method == http.MethodPost:
			mock.blobUploadStarted = true
			w.Header().Set("Location", "/v2/test-repo/blobs/uploads/uuid")
			w.WriteHeader(http.StatusAccepted)

		case strings.HasPrefix(path, "/v2/test-repo/blobs/uploads/") && r.Method == http.MethodPut:
			w.Header().Set("Docker-Content-Digest", "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a")
			w.WriteHeader(http.StatusCreated)

		case strings.HasPrefix(path, "/v2/test-repo/blobs/sha256:") && r.Method == http.MethodHead:
			w.Header().Set("Content-Length", "2")
			w.Header().Set("Docker-Content-Digest", strings.TrimPrefix(path, "/v2/test-repo/blobs/"))
			w.WriteHeader(http.StatusOK)

		default:
			t.Logf("Unexpected request: %s %s", r.Method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(mock.server.Close)

	return mock
}

func (m *mockRegistry) client(t *testing.T) *remote.Registry {
	t.Helper()

	registry, err := remote.NewRegistry(strings.TrimPrefix(m.server.URL, "http://"))
	require.NoError(t, err)
	registry.PlainHTTP = true

	return registry
}

func TestDeleteTag(t *testing.T) {
	t.Parallel()

	t.Run("deletes the manifest when the tag is the last one", func(t *testing.T) {
		t.Parallel()

		mock := newMockRegistry(t, map[string]string{"v1.0.0": manifestDigestA})

		err := DeleteTag(mock.client(t), "test-repo", "v1.0.0")
		require.NoError(t, err)

		assert.Equal(t, []string{manifestDigestA}, mock.deletedRefs,
			"the manifest itself should be deleted, leaving no untagged image behind")
		assert.Empty(t, mock.pushedManifestTags, "no dummy manifest should be pushed")
		assert.False(t, mock.blobUploadStarted, "no dummy config blob should be pushed")
	})

	t.Run("deletes the manifest when sibling tags point to other manifests", func(t *testing.T) {
		t.Parallel()

		mock := newMockRegistry(t, map[string]string{"v1.0.0": manifestDigestA, "latest": manifestDigestB})

		err := DeleteTag(mock.client(t), "test-repo", "v1.0.0")
		require.NoError(t, err)

		assert.Equal(t, []string{manifestDigestA}, mock.deletedRefs,
			"only the manifest of the deleted tag should be removed")
		assert.Empty(t, mock.pushedManifestTags, "no dummy manifest should be pushed")
	})

	t.Run("only untags when a sibling tag shares the manifest", func(t *testing.T) {
		t.Parallel()

		mock := newMockRegistry(t, map[string]string{"v1.0.0": manifestDigestA, "latest": manifestDigestA})

		err := DeleteTag(mock.client(t), "test-repo", "v1.0.0")
		require.NoError(t, err)

		_, dummyBytes, err := generateMinimalManifest()
		require.NoError(t, err)
		dummyDigest := digest.FromBytes(dummyBytes).String()

		assert.Equal(t, []string{dummyDigest}, mock.deletedRefs,
			"only the dummy manifest should be deleted, keeping the shared manifest alive")
		assert.True(t, mock.blobUploadStarted, "the dummy config blob should be pushed")

		var dummyPushed, tagRetargeted bool
		for _, tag := range mock.pushedManifestTags {
			switch {
			case strings.HasPrefix(tag, "__portainer_dummy_"):
				dummyPushed = true
			case tag == "v1.0.0":
				tagRetargeted = true
			}
		}
		assert.True(t, dummyPushed, "a dummy manifest should be pushed")
		assert.True(t, tagRetargeted, "the deleted tag should be retargeted to the dummy manifest")
	})

	t.Run("fails on an unknown tag", func(t *testing.T) {
		t.Parallel()

		mock := newMockRegistry(t, map[string]string{"v1.0.0": manifestDigestA})

		err := DeleteTag(mock.client(t), "test-repo", "missing")
		require.Error(t, err)
		assert.Empty(t, mock.deletedRefs)
	})
}

func TestFilterRepositoriesWithTags(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/repo-a/tags/list":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"repo-a","tags":["v1.0.0"]}`))
		case "/v2/repo-b/tags/list":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"repo-b","tags":[]}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	registry, err := remote.NewRegistry(strings.TrimPrefix(ts.URL, "http://"))
	require.NoError(t, err)
	registry.PlainHTTP = true

	// repo-c errors out and must be treated as empty instead of failing the catalog
	filtered, err := FilterRepositoriesWithTags(context.Background(), registry, []string{"repo-a", "repo-b", "repo-c"})
	require.NoError(t, err)
	assert.Equal(t, []string{"repo-a"}, filtered)
}

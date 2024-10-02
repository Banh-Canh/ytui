package youtube

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/Banh-Canh/ytui/pkg/utils"
)

func TestMain(m *testing.M) {
	// Mock zap.Logger to avoid breaking tests.
	utils.Logger = zap.NewNop()

	// Run the tests
	m.Run()
}

var mockSearchResponse = `[
	{
		"type": "video",
		"title": "Test Video",
		"videoId": "1234",
		"author": "Test Author",
		"authorId": "author123",
		"authorUrl": "https://youtube.com/test_author",
		"videoThumbnails": [
			{
				"quality": "high",
				"url": "https://example.com/thumb.jpg",
				"width": 1280,
				"height": 720
			}
		],
		"description": "This is a test video",
		"viewCount": 1000,
		"published": 1633024800,
		"publishedText": "2 days ago",
		"lengthSeconds": 120
	}
]`

var mockSubscribedVideosResponse = `{
	"videos": [
		{
			"type": "video",
			"title": "Test Video 1",
			"videoId": "video1",
			"author": "Author 1",
			"authorId": "author1",
			"authorUrl": "http://example.com/author1",
			"viewCount": 1000,
			"viewCountText": "1K views",
			"published": 1609459200,
			"publishedText": "1 year ago",
			"lengthSeconds": 360,
			"vieweddate": 0
		},
		{
			"type": "video",
			"title": "Test Video 2",
			"videoId": "video2",
			"author": "Author 2",
			"authorId": "author2",
			"authorUrl": "http://example.com/author2",
			"viewCount": 2000,
			"viewCountText": "2K views",
			"published": 1612137600,
			"publishedText": "11 months ago",
			"lengthSeconds": 480,
			"vieweddate": 0
		}
	]
}`

func mockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestSearchVideos(t *testing.T) {
	// Setup a mock server that returns mockSearchResponse when requested
	ts := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockSearchResponse))
	})
	defer ts.Close()

	// Call searchVideos with the mock server's URL
	proxyURL := ""
	invidiousInstance := ts.URL // Strip out the http:// part
	results, err := searchVideos("test_query", invidiousInstance, proxyURL)

	// Assert no errors
	assert.NoError(t, err)

	// Assert results are not nil
	assert.NotNil(t, results)
	assert.Len(t, *results, 5) // five result because the function looks for the 5 first pages's result

	// Check content of the results
	video := (*results)[0]
	assert.Equal(t, "Test Video", video.Title)
	assert.Equal(t, "1234", video.VideoID)
	assert.Equal(t, "Test Author", video.Author)
}

func TestMakeRequestWithProxy(t *testing.T) {
	// Mock the proxy server and actual API request
	ts := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockSearchResponse))
	})
	defer ts.Close()

	// Proxy URL - for the sake of testing, we can use HTTP instead of SOCKS5
	proxyURL := ts.URL
	targetURL := ts.URL

	resp, err := makeRequestWithProxy(targetURL, proxyURL)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Assert that we received a response
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Test Video")
}

func TestProcessResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser((io.NopCloser(strings.NewReader(mockSearchResponse)))),
	}

	results, err := processResponse(resp, "https://example.com")
	assert.NoError(t, err)

	// Assert the results
	assert.Len(t, *results, 1)
	video := (*results)[0]
	assert.Equal(t, "Test Video", video.Title)
}

func TestSearchVideoInfo(t *testing.T) {
	ts := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"title": "Test Video Info",
			"description": "Test description",
			"channelTitle": "Test Channel",
			"publishedAt": "2023-01-01T00:00:00Z"
		}`))
	})
	defer ts.Close()

	proxyURL := ""
	invidiousInstance := ts.URL
	videoInfo, err := SearchVideoInfo("12345", invidiousInstance, proxyURL)
	assert.NoError(t, err)

	// Check the values returned
	assert.Equal(t, "Test Video Info", videoInfo.Title)
	assert.Equal(t, "Test Channel", videoInfo.Author)
	assert.Equal(t, "Test description", videoInfo.Description)
}

func TestSearchAuthorInfo(t *testing.T) {
	ts := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"author": "Test Channel",
			"authorUrl": "https://youtube.com/test_channel"
		}`))
	})
	defer ts.Close()

	proxyURL := ""
	invidiousInstance := ts.URL
	authorInfo, err := SearchAuthorInfo("channelId123", invidiousInstance, proxyURL)
	assert.NoError(t, err)

	// Check the values returned
	assert.Equal(t, "Test Channel", authorInfo.Author)
	assert.Equal(t, "https://youtube.com/test_channel", authorInfo.AuthorUrl)
}

// Test function for processSubscribedVideoResponse
func TestProcessSubscribedVideoResponse(t *testing.T) {
	// Mock response body
	responseBody := bytes.NewBufferString(mockSubscribedVideosResponse)

	// Create a mock HTTP response
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(responseBody),
	}

	// Call processSubscribedVideoResponse with mock response and URL
	result, err := processSubscribedVideoResponse(mockResponse, "http://example.com")

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, *result, 2)

	// Check individual video entries
	video1 := (*result)[0]
	assert.Equal(t, "Test Video 2", video1.Title)
	assert.Equal(t, "video2", video1.VideoID)
	assert.Equal(t, "Author 2", video1.Author)
	assert.Equal(t, int64(1612137600), video1.Published)

	video2 := (*result)[1]
	assert.Equal(t, "Test Video 1", video2.Title)
	assert.Equal(t, "video1", video2.VideoID)
	assert.Equal(t, "Author 1", video2.Author)
	assert.Equal(t, int64(1609459200), video2.Published)
}

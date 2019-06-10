package spotx

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "spotxtests", NewBidder(new(http.Client), "http://search.spotxchange.com/openrtb"))
}

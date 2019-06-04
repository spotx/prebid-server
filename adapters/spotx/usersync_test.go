package spotx

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestNewSpotxSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//sync.search.spotxchange.com/partner?adv_id=BUYER_ID&uid=BUYER_USER_ID&img=IMG&redir=REDIR_URL"))
	syncer := NewSpotxSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "//sync.search.spotxchange.com/partner?adv_id=BUYER_ID&uid=BUYER_USER_ID&img=IMG&redir=REDIR_URL", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 165, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}

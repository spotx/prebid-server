package spotx

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSpotxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("spotx", 165, temp, adapters.SyncTypeRedirect)
}

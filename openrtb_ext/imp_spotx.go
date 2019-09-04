package openrtb_ext

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// ExtImpSpotX defines the contract for bidrequest.imp[i].ext.spotx
// ORTBVersion refers the Open RTB contract version and is optional, but will default to 2.3
// ChannelID refers to the publisher channel and is required
type ExtImpSpotX struct {
	Boxing      ExtImpSpotXNullBool  `json:"boxing"`
	ORTBVersion string               `json:"ortb_version"`
	ChannelID   uint32               `json:"channel_id"`
	WhiteList   ExtImpSpotXWhiteList `json:"white_list,omitempty"`
	BlackList   ExtImpSpotXBlackList `json:"black_list,omitempty"`
	PriceFloor  float64              `json:"price_floor"`
	Currency    string               `json:"currency"`
	KVP         []ExtImpSpotXKeyVal  `json:"kvp,omitempty"`
}

type ExtImpSpotXWhiteList struct {
	Language []string `json:"lang,omitempty"`
	Seat     []string `json:"seat,omitempty"`
}

type ExtImpSpotXBlackList struct {
	Advertiser []string `json:"advertiser,omitempty"`
	Category   []string `json:"cat,omitempty"`
	Seat       []string `json:"seat,omitempty"`
}

// ExtImpAppnexusKeyVal defines the contract for bidrequest.imp[i].ext.appnexus.keywords[i]
type ExtImpSpotXKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

// NullBool represents a bool that may be null.
// NullBool implements the Scanner interface so
// it can be used as a scan destination, similar to NullString.
type ExtImpSpotXNullBool struct {
	Bool  bool
	Valid bool // Valid is true if Bool is not NULL
}

// Scan implements the Scanner interface.
func (b *ExtImpSpotXNullBool) UnmarshalJSON(data []byte) error {
	var err error
	var v interface{}
	if err = json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case bool:
		b.Bool = x
	case map[string]interface{}:
		err = json.Unmarshal(data, &b.Bool)
	case nil:
		b.Valid = false
		return nil
	default:
		err = fmt.Errorf("json: cannot unmarshal %v into Go value of type null.Bool", reflect.TypeOf(v).Name())
	}
	b.Valid = err == nil
	return err
}

func (b ExtImpSpotXNullBool) MarshalJSON() ([]byte, error) {
	if !b.Valid {
		return []byte("null"), nil
	}
	if !b.Bool {
		return []byte("false"), nil
	}
	return []byte("true"), nil
}

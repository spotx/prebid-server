package spotx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)
// 79391
var (
	ortbVersionMap = map[string]string{
		"" : "2.3",
		"2.3": "2.3",
		"2.5": "2.5",
	}
	skipTrue = int8(1)
)

func parseParam(paramsInput json.RawMessage) (config openrtb_ext.ExtImpSpotX, err error) {
	if err = json.Unmarshal(paramsInput, &config); err != nil {
		return
	}

	if config.ChannelID == 0 {
		return config, errors.New("invalid channel id")
	}

	if v, ok := ortbVersionMap[config.ORTBVersion]; ok {
		config.ORTBVersion = v
	} else  {
		return config, errors.New("unsupported Open RTB version")
	}

	return
}

// getMediaTypeForImp figures out which media type this bid is for.
func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Audio != nil {
				mediaType = openrtb_ext.BidTypeAudio
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}

func parseVastResponse(vastResponse string, bidResponse *adapters.TypedBid) error {

	return nil
}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid *spotxBidExt) (openrtb_ext.BidType, error) {
	switch bid.Width {
	case 0:
		return openrtb_ext.BidTypeBanner, nil
	case 1:
		return openrtb_ext.BidTypeVideo, nil
	case 2:
		return openrtb_ext.BidTypeAudio, nil
	case 3:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unrecognized bid_ad_type in response from appnexus: %d", bid.Width)
	}
}

type spotxReqExt struct {
	Spotx *openrtb_ext.ExtImpSpotX `json:"spotx,omitempty"`
}

type spotxBidExt struct {
	BidID string `json:"bid_id"`
	Code string `json:"code"`
	CreativeID string `json:"creative_id"`
	MediaType string `json:"media_type"`
	Price float64 `json:"price"`
	ADM string `json:"adm"`
	Width int `json:"width"`
	Height int `json:"height"`
	ResponseTime int `json:"response_time_ms"`
}

func kvpToExt(items []openrtb_ext.ExtImpSpotXKeyVal) json.RawMessage {
	result := map[string][]string{

	}
	for _, kvp := range items {
		result[kvp.Key] = kvp.Values
	}
	data, err := json.Marshal(map[string]map[string][]string{"custom": result})
	if err != nil {
		return nil
	}
	return data
}

type SpotxAdapter struct {
	http *adapters.HTTPAdapter
	URI string
	
}

func (a *SpotxAdapter) makeOpenRTBRequest(ctx context.Context, ortbReq *openrtb.BidRequest, param *openrtb_ext.ExtImpSpotX, isDebug bool, ) (*openrtb.BidResponse, *pbs.BidderDebug, error) {

	reqJSON, err := json.Marshal(ortbReq)
	if err != nil {
		return nil, nil, err
	}

	uri := a.getURL(param)

	debug := &pbs.BidderDebug{
		RequestURI: uri,
	}

	if isDebug {
		debug.RequestBody = string(reqJSON)
	}

	httpReq, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	resp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, debug, err
	}

	debug.StatusCode = resp.StatusCode

	if resp.StatusCode == 204 {
		return nil, debug, nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, debug, err
	}
	responseBody := string(body)

	if resp.StatusCode == http.StatusBadRequest {
		return nil, debug, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status %d; body: %s", resp.StatusCode, responseBody),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, debug, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status %d; body: %s", resp.StatusCode, responseBody),
		}
	}

	if isDebug {
		debug.ResponseBody = responseBody
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, debug, err
	}
	return &bidResp, debug, nil
}

func (a *SpotxAdapter) getURL(param *openrtb_ext.ExtImpSpotX) string {
	return fmt.Sprintf("%s/%s/%d", a.URI, param.ORTBVersion, param.ChannelID)
}

// Name must be identical to the BidderName.
func (a *SpotxAdapter) Name() string {
	return string(openrtb_ext.BidderSpotx)
}

// Determines whether this adapter should get callouts if there is not a synched user ID.
func (a *SpotxAdapter) SkipNoCookies() bool {
	return false
}

// Call produces bids which should be considered, given the auction params.
//
// In practice, implementations almost always make one call to an external server here.
// However, that is not a requirement for satisfying this interface.
//
// An error here will cause all bids to be ignored. If the error was caused by bad user input,
// this should return a BadInputError. If it was caused by bad server behavior
// (e.g. 500, unexpected response format, etc), this should return a BadServerResponseError.
func (a *SpotxAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	ortbReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), supportedMediaTypes)

	if err != nil {
		return nil, err
	}

	var param openrtb_ext.ExtImpSpotX

	for i, unit := range bidder.AdUnits {
		param, err = parseParam(unit.Params)
		if err != nil {
			return nil, err
		}

		ortbReq.Imp[i].ID = unit.BidID
		ortbReq.Imp[i].BidFloor = param.PriceFloor
		ortbReq.Imp[i].BidFloorCur = param.Currency

		if param.Boxing.Valid && param.Boxing.Bool {
			ortbReq.Imp[1].Video.BoxingAllowed = int8(1)
		}

		if len(param.KVP) > 0 {
			ortbReq.Imp[i].Ext = kvpToExt(param.KVP)
		}
	}

	ortbReq.BAdv = param.BlackList.Advertiser
	ortbReq.BCat = param.BlackList.Category

	ortbReq.WLang = param.WhiteList.Language

	if len(param.WhiteList.Seat) > 0 {
		ortbReq.WSeat = param.WhiteList.Seat
	} else if len(param.BlackList.Seat) > 0 {
		ortbReq.BSeat = param.BlackList.Seat
	}

	bidResp, debug, err := a.makeOpenRTBRequest(ctx, &ortbReq, &param, req.IsDebug)
	if req.IsDebug {
		bidder.Debug = append(bidder.Debug, debug)
	}
	if err != nil {
		return nil, err
	}

	bids := make(pbs.PBSBidSlice, 0)
	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			numBids++

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
			}

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
				Adm:         bid.AdM,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
			}

			mediaType := getMediaTypeForImp(bid.ImpID, ortbReq.Imp)
			pbid.CreativeMediaType = string(mediaType)

			bids = append(bids, &pbid)
		}
	}

	return bids, nil
}


// MakeRequests makes the HTTP requests which should be made to fetch bids.
//
// Bidder implementations can assume that the incoming BidRequest has:
//
//   1. Only {Imp.Type, Platform} combinations which are valid, as defined by the static/bidder-info.{bidder}.yaml file.
//   2. Imp.Ext of the form {"bidder": params}, where "params" has been validated against the static/bidder-params/{bidder}.json JSON Schema.
//
// nil return values are acceptable, but nil elements *inside* those slices are not.
//
// The errors should contain a list of errors which explain why this bidder's bids will be
// "subpar" in some way. For example: the request contained ad types which this bidder doesn't support.
//
// If the error is caused by bad user input, return an errortypes.BadInput.
func (a *SpotxAdapter) MakeRequests(request *openrtb.BidRequest) (result []*adapters.RequestData, errs []error) {
	if len(request.Ext) > 0 {
		var ext spotxReqExt
		if err := json.Unmarshal(request.Ext, &ext); err == nil {
			uri := a.getURL(ext.Spotx)

			headers := http.Header{}
			headers.Add("Content-Type", "application/json;charset=utf-8")
			headers.Add("Accept", "application/json")

			reqJSON, err := json.Marshal(request)
			if err != nil {
				errs = append(errs, err)
				return
			}

			result = []*adapters.RequestData{{
				Method:  "POST",
				Uri:     uri,
				Body:    reqJSON,
				Headers: headers,
			}}
		} else {
			errs = append(errs,err)
		}
	} else {
		errs = append(errs, errors.New("no extension data found"))
	}
	return
}

// MakeBids unpacks the server's response into Bids.
//
// The bids can be nil (for no bids), but should not contain nil elements.
//
// The errors should contain a list of errors which explain why this bidder's bids will be
// "subpar" in some way. For example: the server response didn't have the expected format.
//
// If the error was caused by bad user input, return a errortypes.BadInput.
// If the error was caused by a bad server response, return a errortypes.BadServerResponse
func (a *SpotxAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	var bidResponse *adapters.TypedBid
	bidResponses := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid))

	var errs []error
	var cat string
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if len(bid.Cat) > 0 {
				cat = bid.Cat[0]
			} else {
				cat = ""
			}
			fmt.Printf("%+v\n", bid)
			bidResponse = &adapters.TypedBid{
				BidType: openrtb_ext.BidTypeVideo,
				Bid:      &bid,
				BidVideo: &openrtb_ext.ExtBidPrebidVideo{
					Duration: 0,
					PrimaryCategory: cat,
				},
			}

			if bid.AdM != "" {
				if err := parseVastResponse(bid.AdM, bidResponse); err != nil {
					errs = append(errs, err)
				}
			}

			bidResponses.Bids = append(bidResponses.Bids, bidResponse)
		}
	}
	return bidResponses, errs
}

func NewAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *SpotxAdapter {
	return NewBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

func NewBidder(client *http.Client, endpoint string) *SpotxAdapter {
	a := &adapters.HTTPAdapter{Client: client}


	return &SpotxAdapter{
		http:           a,
		URI:            endpoint,
	}
}

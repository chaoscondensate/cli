package ots

const PublicProfileID = "opentimestamps-public-v1"

type CalendarSource struct {
	ID                 string   `json:"id"`
	SubmissionEndpoint string   `json:"submission_endpoint"`
	AcceptedIdentities []string `json:"accepted_identities"`
}

type BitcoinSource struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

type PublicProfile struct {
	ID                    string           `json:"id"`
	Calendars             []CalendarSource `json:"calendars"`
	CalendarMinimum       int              `json:"calendar_minimum"`
	BitcoinSources        []BitcoinSource  `json:"bitcoin_sources"`
	MaximumUniqueHeights  int              `json:"maximum_unique_heights"`
	MaximumHTTPRequests   int              `json:"maximum_http_requests"`
	MaximumConcurrentHTTP int              `json:"maximum_concurrent_http"`
	Experimental          bool             `json:"experimental"`
	TrustLimitation       string           `json:"trust_limitation"`
	PrivacyLimitation     string           `json:"privacy_limitation"`
}

func Profile() PublicProfile {
	return PublicProfile{
		ID: PublicProfileID,
		Calendars: []CalendarSource{
			{ID: "ots-calendar-a", SubmissionEndpoint: "https://a.pool.opentimestamps.org", AcceptedIdentities: []string{"https://alice.btc.calendar.opentimestamps.org"}},
			{ID: "ots-calendar-b", SubmissionEndpoint: "https://b.pool.opentimestamps.org", AcceptedIdentities: []string{"https://bob.btc.calendar.opentimestamps.org"}},
			{ID: "ots-calendar-eternitywall", SubmissionEndpoint: "https://a.pool.eternitywall.com", AcceptedIdentities: []string{"https://finney.calendar.eternitywall.com"}},
			{ID: "ots-calendar-catallaxy", SubmissionEndpoint: "https://ots.btc.catallaxy.com", AcceptedIdentities: []string{"https://btc.calendar.catallaxy.com", "https://ots.btc.catallaxy.com"}},
		},
		CalendarMinimum:       2,
		BitcoinSources:        []BitcoinSource{{ID: "mempool-space", Endpoint: "https://mempool.space/api"}, {ID: "blockstream", Endpoint: "https://blockstream.info/api"}},
		MaximumUniqueHeights:  32,
		MaximumHTTPRequests:   128,
		MaximumConcurrentHTTP: 4,
		Experimental:          true,
		TrustLimitation:       "Public verification requires both named services to agree; it still trusts them for canonical-chain selection.",
		PrivacyLimitation:     "Calendar services learn request timing and blinded commitments; Bitcoin services learn requested block heights and approximate timestamp periods.",
	}
}

func acceptedCalendarIdentity(source CalendarSource, identity string) bool {
	for _, accepted := range source.AcceptedIdentities {
		if identity == accepted || identity == accepted+"/" {
			return true
		}
	}
	return false
}

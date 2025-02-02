package proxi

// UserData represents the stored information for intercepted HTTP transactions
// and any associated metadata

type UserData struct {
	RequestMethod   string
	RequestURL      string
	RequestHeaders  map[string][]string
	RequestBody     string
	ResponseCode    int
	ResponseHeaders map[string][]string
	ResponseBody    string
}

// NewUserData initializes a UserData instance from the intercepted request and response transaction
// It can store the serialized request-reply states and full request-response context.

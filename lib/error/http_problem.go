package sebakerror

import (
	"encoding/json"
	"net/http"
)

// Default Http Problem with as-is description
var (
	ProblemDefaultContinue           = NewStatusProblem(http.StatusContinue)
	ProblemDefaultSwitchingProtocols = NewStatusProblem(http.StatusSwitchingProtocols)
	ProblemDefaultProcessing         = NewStatusProblem(http.StatusProcessing)

	ProblemDefaultOK                   = NewStatusProblem(http.StatusOK)
	ProblemDefaultCreated              = NewStatusProblem(http.StatusCreated)
	ProblemDefaultAccepted             = NewStatusProblem(http.StatusAccepted)
	ProblemDefaultNonAuthoritativeInfo = NewStatusProblem(http.StatusNonAuthoritativeInfo)
	ProblemDefaultNoContent            = NewStatusProblem(http.StatusNoContent)
	ProblemDefaultResetContent         = NewStatusProblem(http.StatusResetContent)
	ProblemDefaultPartialContent       = NewStatusProblem(http.StatusPartialContent)
	ProblemDefaultMultiStatus          = NewStatusProblem(http.StatusMultiStatus)
	ProblemDefaultAlreadyReported      = NewStatusProblem(http.StatusAlreadyReported)
	ProblemDefaultIMUsed               = NewStatusProblem(http.StatusIMUsed)

	ProblemDefaultMultipleChoices  = NewStatusProblem(http.StatusMultipleChoices)
	ProblemDefaultMovedPermanently = NewStatusProblem(http.StatusMovedPermanently)
	ProblemDefaultFound            = NewStatusProblem(http.StatusFound)
	ProblemDefaultSeeOther         = NewStatusProblem(http.StatusSeeOther)
	ProblemDefaultNotModified      = NewStatusProblem(http.StatusNotModified)
	ProblemDefaultUseProxy         = NewStatusProblem(http.StatusUseProxy)

	ProblemDefaultTemporaryRedirect = NewStatusProblem(http.StatusTemporaryRedirect)
	ProblemDefaultPermanentRedirect = NewStatusProblem(http.StatusPermanentRedirect)

	ProblemDefaultBadRequest                   = NewStatusProblem(http.StatusBadRequest)
	ProblemDefaultUnauthorized                 = NewStatusProblem(http.StatusUnauthorized)
	ProblemDefaultPaymentRequired              = NewStatusProblem(http.StatusPaymentRequired)
	ProblemDefaultForbidden                    = NewStatusProblem(http.StatusForbidden)
	ProblemDefaultNotFound                     = NewStatusProblem(http.StatusNotFound)
	ProblemDefaultMethodNotAllowed             = NewStatusProblem(http.StatusMethodNotAllowed)
	ProblemDefaultNotAcceptable                = NewStatusProblem(http.StatusNotAcceptable)
	ProblemDefaultProxyAuthRequired            = NewStatusProblem(http.StatusProxyAuthRequired)
	ProblemDefaultRequestTimeout               = NewStatusProblem(http.StatusRequestTimeout)
	ProblemDefaultConflict                     = NewStatusProblem(http.StatusConflict)
	ProblemDefaultGone                         = NewStatusProblem(http.StatusGone)
	ProblemDefaultLengthRequired               = NewStatusProblem(http.StatusLengthRequired)
	ProblemDefaultPreconditionFailed           = NewStatusProblem(http.StatusPreconditionFailed)
	ProblemDefaultRequestEntityTooLarge        = NewStatusProblem(http.StatusRequestEntityTooLarge)
	ProblemDefaultRequestURITooLong            = NewStatusProblem(http.StatusRequestURITooLong)
	ProblemDefaultUnsupportedMediaType         = NewStatusProblem(http.StatusUnsupportedMediaType)
	ProblemDefaultRequestedRangeNotSatisfiable = NewStatusProblem(http.StatusRequestedRangeNotSatisfiable)
	ProblemDefaultExpectationFailed            = NewStatusProblem(http.StatusExpectationFailed)
	ProblemDefaultTeapot                       = NewStatusProblem(http.StatusTeapot)
	ProblemDefaultUnprocessableEntity          = NewStatusProblem(http.StatusUnprocessableEntity)
	ProblemDefaultLocked                       = NewStatusProblem(http.StatusLocked)
	ProblemDefaultFailedDependency             = NewStatusProblem(http.StatusFailedDependency)
	ProblemDefaultUpgradeRequired              = NewStatusProblem(http.StatusUpgradeRequired)
	ProblemDefaultPreconditionRequired         = NewStatusProblem(http.StatusPreconditionRequired)
	ProblemDefaultTooManyRequests              = NewStatusProblem(http.StatusTooManyRequests)
	ProblemDefaultRequestHeaderFieldsTooLarge  = NewStatusProblem(http.StatusRequestHeaderFieldsTooLarge)
	ProblemDefaultUnavailableForLegalReasons   = NewStatusProblem(http.StatusUnavailableForLegalReasons)

	ProblemDefaultInternalServerError           = NewStatusProblem(http.StatusInternalServerError)
	ProblemDefaultNotImplemented                = NewStatusProblem(http.StatusNotImplemented)
	ProblemDefaultBadGateway                    = NewStatusProblem(http.StatusBadGateway)
	ProblemDefaultServiceUnavailable            = NewStatusProblem(http.StatusServiceUnavailable)
	ProblemDefaultGatewayTimeout                = NewStatusProblem(http.StatusGatewayTimeout)
	ProblemDefaultHTTPVersionNotSupported       = NewStatusProblem(http.StatusHTTPVersionNotSupported)
	ProblemDefaultVariantAlsoNegotiates         = NewStatusProblem(http.StatusVariantAlsoNegotiates)
	ProblemDefaultInsufficientStorage           = NewStatusProblem(http.StatusInsufficientStorage)
	ProblemDefaultLoopDetected                  = NewStatusProblem(http.StatusLoopDetected)
	ProblemDefaultNotExtended                   = NewStatusProblem(http.StatusNotExtended)
	ProblemDefaultNetworkAuthenticationRequired = NewStatusProblem(http.StatusNetworkAuthenticationRequired)
)

//  Default Http Problem with new description
var (
	// Example
	ProblemBadRequestNotEnoughParam = NewDetailedStatusProblem(http.StatusBadRequest, "paramaters are not enough")
)

// Http Problem with description and Instance
var (
	// Example
	ProblemBadRequestNotEnoughParamWithInstance = NewDetailedStatusProblem(http.StatusBadRequest, "paramaters are not enough").SetInstance("blahblah")
)

type Problem struct {
	// "type" (string) - A URI reference [RFC3986] that identifies the
	// problem type.  This specification encourages that, when
	// dereferenced, it provide human-readable documentation for the
	// problem type (e.g., using HTML [W3C.REC-html5-20141028]).  When
	// this member is not present, its value is assumed to be
	// "about:blank".
	Type string `json:"type"`

	//"title" (string) - A short, human-readable summary of the problem
	//type.  It SHOULD NOT change from occurrence to occurrence of the
	//problem, except for purposes of localization (e.g., using
	//proactive content negotiation; see [RFC7231], Section 3.4).
	Title string `json:"title"`

	//"status" (number) - The HTTP status code ([RFC7231], Section 6)
	//generated by the origin server for this occurrence of the problem.
	Status int `json:"status,omitempty"`

	//"detail" (string) - A human-readable explanation specific to this
	//occurrence of the problem.
	Detail string `json:"detail,omitempty"`

	//"instance" (string) - A URI reference that identifies the specific
	//occurrence of the problem.  It may or may not yield further
	//information if dereferenced.
	Instance string `json:"instance,omitempty"`
}

func NewStatusProblem(status int) Problem {
	return Problem{Type: "about:blank", Status: status, Detail: http.StatusText(status)}
}

func NewDetailedStatusProblem(status int, detail string) Problem {
	p := NewStatusProblem(status)
	p.Detail = detail
	return p
}

func (p Problem) SetInstance(instance string) Problem {
	p.Instance = instance
	return p
}

func (p Problem) SetDetail(detail string) Problem {
	p.Detail = detail
	return p
}

func (p Problem) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

package subsonic

type apiErrorCode int

// Error codes defined in the Subsonic API documentation.
const (
	errCodeGeneric          apiErrorCode = 0
	errCodeMissingParameter apiErrorCode = 10
	errCodeVersionClient    apiErrorCode = 20
	errCodeVersionServer    apiErrorCode = 30
	errCodeWrongUserOrPass  apiErrorCode = 40
	errCodeTokenAuthLDAP    apiErrorCode = 41
	errCodeNotAuthorized    apiErrorCode = 50
	errCodeNotFound         apiErrorCode = 70
)

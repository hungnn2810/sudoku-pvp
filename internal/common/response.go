package common

// ErrorResponse is the standard JSON error body returned by all API endpoints.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TODO: import gin after go get in Wave 2 and add handler helpers:
//
//   func Error(c *gin.Context, status int, code string, msg string) {
//       c.JSON(status, ErrorResponse{Code: code, Message: msg})
//   }
//
//   func OK(c *gin.Context, data any) {
//       c.JSON(200, data)
//   }

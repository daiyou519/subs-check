package model

type StandardResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type SuccessResponse struct {
	Code    int         `json:"code" example:"200"`
	Message string      `json:"message" example:"success"`
	Data    interface{} `json:"data"`
}

type ServerErrorResponse struct {
	Code    int         `json:"code" example:"500"`
	Message string      `json:"message" example:"Internal server error"`
	Data    interface{} `json:"data"`
}

type UnauthorizedResponse struct {
	Code    int         `json:"code" example:"401"`
	Message string      `json:"message" example:"Unauthorized access"`
	Data    interface{} `json:"data"`
}

type BadRequestResponse struct {
	Code    int         `json:"code" example:"400"`
	Message string      `json:"message" example:"Invalid request parameters"`
	Data    interface{} `json:"data"`
}

type NotFoundResponse struct {
	Code    int         `json:"code" example:"404"`
	Message string      `json:"message" example:"Not found"`
	Data    interface{} `json:"data"`
}

type ConflictResponse struct {
	Code    int         `json:"code" example:"409"`
	Message string      `json:"message" example:"Conflict"`
	Data    interface{} `json:"data"`
}

// Package modeldto provides models for data transfer objects.

package modeldto

type (
	RequestGetByUserID struct {
		UserID string `json:"user_id"`
	}

	ResponseProductCode struct {
		ProductCode string `json:"product_code" example:"upload_23andme_v5_b2c_array_txt"`
	}

	ResponseProcessingStatus struct {
		Status string `json:"current_status" example:"done"`
	}
)

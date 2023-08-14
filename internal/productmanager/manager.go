// Package productmanager provides methods for product code derivation.

package productmanager

import "github.com/rs/zerolog"

// ProductManager defines a new object and sets its attributes.
type ProductManager struct {
	log *zerolog.Logger
}

// NewProductManager initializes a new ProductManager instance.
func NewProductManager(logger *zerolog.Logger) *ProductManager {
	logger.Debug().Msg("calling initializer of product manager service")
	return &ProductManager{log: logger}
}

// GetProductCode returns a product code for a given file mode.
func (p *ProductManager) GetProductCode(mode string) string {
	p.log.Debug().Msg("calling `GetProductCode` method")
	switch mode {
	case "vcf":
		return "upload_genotek_b2c_array_vcf"
	case "tsv":
		return "upload_23andme_v5_b2c_array_txt"
	case "csv":
		return "upload_23andme_v5_b2c_array_txt"
	default:
		return "upload_23andme_v5_b2c_array_txt"
	}
}

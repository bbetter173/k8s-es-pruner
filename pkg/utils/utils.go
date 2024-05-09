package utils

import (
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

// ParseSize converts a size string to bytes.
func ParseSize(sizeStr string) (int64, error) {
	units := map[string]int64{
		"KB":  1000,
		"MB":  1000 * 1000,
		"GB":  1000 * 1000 * 1000,
		"TB":  1000 * 1000 * 1000 * 1000,
		"KIB": 1024,
		"MIB": 1024 * 1024,
		"GIB": 1024 * 1024 * 1024,
		"TIB": 1024 * 1024 * 1024 * 1024,
	}

	// Remove all whitespaces and convert to uppercase for standard comparison
	sizeStr = strings.ToUpper(strings.ReplaceAll(sizeStr, " ", ""))

	// Find which unit is used
	var unit string
	var numberStr string
	for k := range units {
		if strings.HasSuffix(sizeStr, k) {
			unit = k
			numberStr = strings.TrimSuffix(sizeStr, k)
			break
		}
	}

	if unit == "" {
		return 0, fmt.Errorf("invalid size unit in '%s'", sizeStr)
	}

	// Parse the number part
	number, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number format in '%s': %v", sizeStr, err)
	}

	// Calculate the total number of bytes
	totalBytes := int64(number * float64(units[unit]))

	return totalBytes, nil
}

func SetupLogger() *zap.SugaredLogger {
	logger, _ := zap.NewProduction()

	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			fmt.Printf("error syncing logger: %v", err)
		}
	}(logger) // flushes buffer, if any
	return logger.Sugar()
}

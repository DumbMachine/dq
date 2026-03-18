package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dumbmachine/db-cli/pkg/types"
)

func ExitWithError(errType, message, suggestion string, exitCode int) {
	resp := types.ErrorResponse{
		Error:      errType,
		Message:    message,
		Suggestion: suggestion,
		ExitCode:   exitCode,
	}
	data, _ := json.Marshal(resp)
	fmt.Fprintln(os.Stderr, string(data))
	os.Exit(exitCode)
}

package cons

import (
	"os"

	"github.com/bronystylecrazy/ultrastructure/di"
)

func OS() di.Node {
	return di.Options(
		di.Provide(os.Hostname, di.As[HostName]()),
		di.Provide(os.UserHomeDir, di.As[HomeDir]()),
	)
}

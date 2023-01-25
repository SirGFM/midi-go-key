package err_wrap

import (
	"fmt"
)

func Wrap(base, wrapped error) error {
	return fmt.Errorf(base.Error()+": %w", wrapped)
}

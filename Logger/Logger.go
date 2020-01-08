// Logger
package Logger

import (
	"fmt"
	"time"
)

//Log writes to CommandLine and SysLogger Server messages
//args are the arguments for log
func Log(args ...interface{}) {
	timeNow := time.Now()
	fmt.Println(timeNow, " : ", args)
}

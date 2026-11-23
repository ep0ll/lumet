// Package progress defines a UI-agnostic build progress event contract.
package progress
import("context";"time";ocispecs "github.com/opencontainers/image-spec/specs-go/v1")
type Phase string
const(Started Phase="started";Completed Phase="completed";Failed Phase="failed")
type Event struct{Phase Phase;Group,Message,Instruction string;InstructionIndex,InstructionTotal int;StageName string;StageIndex,StageTotal int;Platform ocispecs.Platform;PlatformSpecified bool;StartedAt,FinishedAt time.Time;Err error}
type Reporter interface{Report(context.Context,Event)error}
type ReporterFunc func(context.Context,Event)error
func(f ReporterFunc)Report(c context.Context,e Event)error{return f(c,e)}

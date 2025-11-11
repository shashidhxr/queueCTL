package queue

type State string

const (
	StatePending State = "pending"
	StateProcessing State = "processing"
	StateCompleted State = "completed"
	StateFailed State = "failed"
	StateDead State = "dead"
)

type Job struct {
	Id			string		`json:"id" db:"id"`
	Command		string 		`json:"command" db:"command"`
	State		string 		`json:"state" db:"state"`
	Attempts	string		`json:"attempts db:"attempts"`
	MaxRetries	string 		`json:"max_retries" db:"max_retires"`
	AvailabeAt	string 		`json:"-" db:"available_at"`
	CreatedAt	string 		`json:"created_at" db:"created_at"`
	updatedAt 	string 		`json:"updated_at" db:"updated_at"`
}
package clock

import "github.com/google/uuid"

type UUIDv7Generator struct{}

func (UUIDv7Generator) NewID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic("clock: failed to generate UUIDv7: " + err.Error())
	}
	return id
}

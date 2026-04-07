package engine

import "dev-null/internal/domain"

// clampMIDI clamps a value to the standard MIDI range 0-127.
func clampMIDI(v int) int {
	if v < 0 {
		return 0
	}
	if v > 127 {
		return 127
	}
	return v
}

// clampChannel clamps a MIDI channel to 0-15.
func clampChannel(v int) int {
	if v < 0 {
		return 0
	}
	if v > 15 {
		return 15
	}
	return v
}

// newNoteOnEvent creates a NoteOn MIDI event.
func newNoteOnEvent(channel, note, velocity, durationMs int) domain.MidiEvent {
	return domain.MidiEvent{
		Type:       domain.MidiNoteOn,
		Channel:    clampChannel(channel),
		Note:       clampMIDI(note),
		Velocity:   clampMIDI(velocity),
		DurationMs: durationMs,
	}
}

// newProgramChangeEvent creates a ProgramChange MIDI event.
func newProgramChangeEvent(channel, program int) domain.MidiEvent {
	return domain.MidiEvent{
		Type:    domain.MidiProgramChange,
		Channel: clampChannel(channel),
		Program: clampMIDI(program),
	}
}

// newControlChangeEvent creates a ControlChange MIDI event.
func newControlChangeEvent(channel, controller, value int) domain.MidiEvent {
	return domain.MidiEvent{
		Type:       domain.MidiControlChange,
		Channel:    clampChannel(channel),
		Controller: clampMIDI(controller),
		Velocity:   clampMIDI(value),
	}
}

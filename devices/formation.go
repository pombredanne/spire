package devices

import "sync"

type stateMap map[string]interface{}

// deviceID -> device state
type deviceStateMap map[string]stateMap

type formationS struct {
	state   stateMap
	devices deviceStateMap
}

// FormationMap ...
type FormationMap struct {
	m map[string]formationS
	d map[string]string // device name -> formation ID
	l sync.RWMutex
}

// NewFormationMap ...
func NewFormationMap() *FormationMap {
	return &FormationMap{
		m: make(map[string]formationS),
		d: make(map[string]string),
	}
}

// Lock ...
func (fm *FormationMap) Lock() {
	fm.l.Lock()
}

// Unlock ...
func (fm *FormationMap) Unlock() {
	fm.l.Unlock()
}

// RLock ...
func (fm *FormationMap) RLock() {
	fm.l.RLock()
}

// RUnlock ...
func (fm *FormationMap) RUnlock() {
	fm.l.RUnlock()
}

// PutState ...
func (fm *FormationMap) PutState(formationID, key string, value interface{}) {
	formation, exists := fm.m[formationID]

	if exists {
		formation.state[key] = value
	} else {
		formation = formationS{
			state:   stateMap{key: value},
			devices: make(deviceStateMap),
		}

		fm.m[formationID] = formation
	}
}

// GetState ...
func (fm *FormationMap) GetState(formationID, key string) interface{} {
	formation, exists := fm.m[formationID]

	if !exists {
		return nil
	}

	return formation.state[key]
}

// PutDeviceState ...
func (fm *FormationMap) PutDeviceState(formationID, deviceName, key string, value interface{}) {
	formation, fExists := fm.m[formationID]

	if !fExists {
		formation = formationS{make(stateMap), make(deviceStateMap)}
		fm.m[formationID] = formation
	}

	state, dExists := formation.devices[deviceName]
	if !dExists {
		state = make(stateMap)
		formation.devices[deviceName] = state
	}

	state[key] = value
	fm.d[deviceName] = formationID
}

// GetDeviceState ...
func (fm *FormationMap) GetDeviceState(deviceName, key string) interface{} {
	if formationID, exists := fm.d[deviceName]; exists {

		if formation, exists := fm.m[formationID]; exists {

			if state, exists := formation.devices[deviceName]; exists {
				return state[key]
			}
		}
	}

	return nil
}

// DeleteDeviceState ...
func (fm *FormationMap) DeleteDeviceState(formationID, deviceName, key string) {
	formation, fExists := fm.m[formationID]

	if !fExists {
		return
	}

	state, dExists := formation.devices[deviceName]
	if dExists {
		delete(state, key)
	}

	delete(fm.d, deviceName)
}

// FormationID returns the devices formation ID
func (fm *FormationMap) FormationID(deviceName string) string {
	return fm.d[deviceName]
}

// AddDevice ...
func (fm *FormationMap) AddDevice(deviceName, formationID string) {
	fm.d[deviceName] = formationID
}

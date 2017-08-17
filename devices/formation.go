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
	l sync.RWMutex
}

// NewFormationMap ...
func NewFormationMap() *FormationMap {
	return &FormationMap{
		m: make(map[string]formationS),
	}
}

func (fm *FormationMap) get(formationID string) formationS {
	fm.l.RLock()
	defer fm.l.RUnlock()
	return fm.m[formationID]
}

// PutState ...
func (fm *FormationMap) PutState(formationID, key string, value interface{}) {
	fm.l.Lock()
	defer fm.l.Unlock()

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
	fm.l.RLock()
	defer fm.l.RUnlock()

	formation, exists := fm.m[formationID]

	if !exists {
		return nil
	}

	return formation.state[key]
}

// PutDeviceState ...
func (fm *FormationMap) PutDeviceState(formationID, deviceName, key string, value interface{}) {
	fm.l.Lock()
	defer fm.l.Unlock()

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
}

// GetDeviceState returns the value from the device state for the given key and the devices formationID.
func (fm *FormationMap) GetDeviceState(deviceName, key string) (interface{}, string) {
	fm.l.RLock()
	defer fm.l.RUnlock()

	for formationID, formation := range fm.m {

		state, exists := formation.devices[deviceName]
		if exists {
			return state[key], formationID
		}
	}

	return nil, ""
}

// DeleteDeviceState ...
func (fm *FormationMap) DeleteDeviceState(formationID, deviceName, key string) {
	fm.l.Lock()
	defer fm.l.Unlock()

	formation, fExists := fm.m[formationID]

	if !fExists {
		return
	}

	state, dExists := formation.devices[deviceName]
	if dExists {
		delete(state, key)
	}
}

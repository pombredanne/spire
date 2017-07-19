package devices

// TODO figure out locking

type stateMap map[string]interface{}

// deviceID -> device state
type deviceStateMap map[string]stateMap

type formationS struct {
	state   stateMap
	devices deviceStateMap
}

type formationMap map[string]formationS

func (fm formationMap) putState(formationID, key string, value interface{}) {
	formation, exists := fm[formationID]

	if exists {
		formation.state[key] = value
	} else {
		formation = formationS{
			state:   stateMap{key: value},
			devices: make(deviceStateMap),
		}

		fm[formationID] = formation
	}
}

func (fm formationMap) getState(formationID, key string) interface{} {
	formation, exists := fm[formationID]

	if !exists {
		return nil
	}

	return formation.state[key]
}

func (fm formationMap) putDeviceState(formationID, deviceName, key string, value interface{}) {
	formation, fExists := fm[formationID]

	if !fExists {
		formation = formationS{make(stateMap), make(deviceStateMap)}
		fm[formationID] = formation
	}

	deviceState, dExists := formation.devices[deviceName]
	if !dExists {
		deviceState = make(stateMap)
		formation.devices[deviceName] = deviceState
	}

	deviceState[key] = value
}

func (fm formationMap) getDeviceState(formationID, deviceName, key string) interface{} {
	formation, fExists := fm[formationID]
	if !fExists {
		return nil
	}

	deviceState, dExists := formation.devices[deviceName]
	if !dExists {
		return nil
	}

	return deviceState[key]
}

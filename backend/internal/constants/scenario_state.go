package constants

type ScenarioState string

const (
	ScenarioDraft     ScenarioState = "draft"
	ScenarioSimulated ScenarioState = "simulated"
	ScenarioReady     ScenarioState = "ready"
	ScenarioExecuting ScenarioState = "executing"
	ScenarioVerified  ScenarioState = "verified"
	ScenarioRollback  ScenarioState = "rollback"
)

func ScenarioStateValues() []ScenarioState {
	return []ScenarioState{ScenarioDraft, ScenarioSimulated, ScenarioReady, ScenarioExecuting, ScenarioVerified, ScenarioRollback}
}

func CanTransitionScenario(from, to ScenarioState) bool {
	allowed := map[ScenarioState][]ScenarioState{
		ScenarioDraft:     {ScenarioSimulated},
		ScenarioSimulated: {ScenarioReady, ScenarioDraft},
		ScenarioReady:     {ScenarioExecuting, ScenarioDraft},
		ScenarioExecuting: {ScenarioVerified, ScenarioRollback},
	}
	for _, candidate := range allowed[from] {
		if candidate == to {
			return true
		}
	}
	return false
}

// p4infoUtils.go

package p4InfoUtils

import (
	p4_config "p4/config"
	)

func GetTableIdFromName(p4Info *p4_config.P4Info, inTable string) (uint32) {
	for _, table := range p4Info.Tables {
		if (table.Preamble.Name  == inTable) {
			return table.Preamble.Id
		}
	}
	return 0
}

func GetTableNameFromId(p4Info *p4_config.P4Info, inTabId uint32) (string) {
	for _, table := range p4Info.Tables {
		if (table.Preamble.Id  == inTabId) {
			return table.Preamble.Name
		}
	}
	return "__invalid_table_name__"
}

func GetMatchIdInTable(p4Info *p4_config.P4Info, tableId uint32, fldName string) (uint32) {
	for _, table := range p4Info.Tables {
		if (table.Preamble.Id  == tableId) {
			for _, matchFld := range table.MatchFields {
				if matchFld.Name == fldName {
					return matchFld.Id
				}
			}
		}
	}
	return 0
}

func GetMatchNameInTable(p4Info *p4_config.P4Info, tableId uint32, fldId uint32) (string) {
	for _, table := range p4Info.Tables {
		if (table.Preamble.Id  == tableId) {
			for _, matchFld := range table.MatchFields {
				if matchFld.Id == fldId {
					return matchFld.Name
				}
			}
		}
	}
	return "__invalid_match_name__"
}

func GetActionId(p4Info *p4_config.P4Info, actionStr string) (uint32) {
	for _, action := range p4Info.Actions {
		if action.Preamble.Name == actionStr {
			return action.Preamble.Id
		}
	}
	return 0
}

func GetActionName(p4Info *p4_config.P4Info, actionId uint32) (string) {
	for _, action := range p4Info.Actions {
		if action.Preamble.Id == actionId {
			return action.Preamble.Name
		}
	}
	return "__invalid_action_name__"
}

func GetParamIdInAction(p4Info *p4_config.P4Info, actionId uint32, paramStr string) (uint32) {
	for _, action := range p4Info.Actions {
		if action.Preamble.Id == actionId {
			for _, param := range action.Params {
				if param.Name == paramStr {
					return param.Id
				}
			}
		}
	}
	return 0
}

func GetParamNameInAction(p4Info *p4_config.P4Info, actionId uint32, paramId uint32) (string) {
	for _, action := range p4Info.Actions {
		if action.Preamble.Id == actionId {
			for _, param := range action.Params {
				if param.Id == paramId {
					return param.Name
				}
			}
		}
	}
	return "__invalid_param_name__"
}

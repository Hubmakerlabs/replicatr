package agent

func (b *Backend) AddUser(pubKey string) (err error) {
	methodName := "add_user"
	args := []any{pubKey}
	var result string
	err = b.Agent.Query(b.CanisterID, methodName, args, []any{&result})
	if err != nil {
		return
	} else if result != "success" {
		err = log.E.Err("failed to add user")
		return
	}
	return nil
}

func (b *Backend) RemoveUser(pubKey string, perm bool) (err error) {
	methodName := "remove_user"
	args := []any{pubKey, perm}
	var result string
	err = b.Agent.Query(b.CanisterID, methodName, args, []any{&result})
	if err != nil {
		return
	} else if result != "success" {
		err = log.E.Err("failed to remove user")
		return
	}
	return nil
}

func (b *Backend) GetPermission(pubKey string) (err error) {
	methodName := "get_permission"
	args := []any{pubKey}
	var result string
	err = b.Agent.Query(b.CanisterID, methodName, args, []any{&result})
	if err != nil {
		return
	} else if result != "success" {
		err = log.E.Err("failed to get permission")
		return
	}
	return nil
}

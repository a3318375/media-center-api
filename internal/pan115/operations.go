package pan115

type FileGateway interface {
	Gateway
	CreateDirectory(parent, name string) (File, error)
	RenameFile(id, name string) error
	MoveFile(id, parent string) error
}

func (c *Client) RenameFile(id, name string) error { return SafeOperation(c.sdk.Rename(id, name)) }
func (c *Client) MoveFile(id, parent string) error { return SafeOperation(c.sdk.Move(parent, id)) }
func SafeOperation(err error) error {
	if err == nil {
		return nil
	}
	return SafeError(err)
}

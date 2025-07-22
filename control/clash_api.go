package control

import (
	"fmt"
	"net"
	"sync"
)

func (c *ControlPlane) AbortConnection(id string) (err error) {
	conn, ok := c.inConnections.Load(id)
	if !ok {
		return fmt.Errorf("connection not found")
	}
	if err = conn.(net.Conn).Close(); err != nil {
		return err
	}
	return nil
}

func (c *ControlPlane) GetAllConnections() *sync.Map {
	return &c.inConnections
}

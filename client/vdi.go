package client

import (
	"errors"
	"fmt"
)

type Disk struct {
	VBD
	VDI
}

type VDI struct {
	VDIId     string   `json:"id"`
	SrId      string   `json:"$SR"`
	NameLabel string   `json:"name_label"`
	Size      int      `json:"size"`
	VBDs      []string `json:"$VBDs"`
}

func (v VDI) Compare(obj interface{}) bool {
	other := obj.(VDI)

	if v.VDIId != "" && other.VDIId == v.VDIId {
		return true
	}

	return false
}

// TODO: Change this file to storage or disks?
type VBD struct {
	Id        string `json:"id"`
	Attached  bool
	Device    string
	ReadOnly  bool   `json:"read_only"`
	VmId      string `json:"VM"`
	VDI       string `json:"VDI"`
	IsCdDrive bool   `json:"is_cd_drive"`
	Position  string
	Bootable  bool
	PoolId    string `json:"$poolId"`
}

func (v VBD) Compare(obj interface{}) bool {
	other := obj.(VBD)
	if v.IsCdDrive != other.IsCdDrive {
		return false
	}

	if other.VmId != "" && v.VmId == other.VmId {
		return true
	}

	return false
}

func (c *Client) GetDisks(vm *Vm) ([]Disk, error) {
	obj, err := c.FindFromGetAllObjects(VBD{VmId: vm.Id, IsCdDrive: false})

	if _, ok := err.(NotFound); ok {
		return []Disk{}, nil
	}

	if err != nil {
		return nil, err
	}
	disks, ok := obj.([]VBD)

	if !ok {
		return []Disk{}, errors.New(fmt.Sprintf("failed to coerce %v into VBD", obj))
	}

	fmt.Printf("Found the following disks before looking for VDIs %+v", disks)
	vdis := []Disk{}
	for _, disk := range disks {
		vdi, err := c.GetParentVDI(disk)

		if err != nil {
			return []Disk{}, err
		}

		vdis = append(vdis, Disk{disk, vdi})
	}
	return vdis, nil
}

func (c *Client) GetParentVDI(vbd VBD) (VDI, error) {
	obj, err := c.FindFromGetAllObjects(VDI{
		VDIId: vbd.VDI,
	})

	// Rather than detect not found errors we let the caller
	// decide that for themselves.
	if err != nil {
		return VDI{}, err
	}
	disks, ok := obj.([]VDI)

	if !ok {
		return VDI{}, errors.New(fmt.Sprintf("failed to coerce %+v into VDI", obj))
	}

	if len(disks) != 1 {
		return VDI{}, errors.New(fmt.Sprintf("expected Vm VDI to only contain a single VBD, instead found %d", len(disks)))
	}
	return disks[0], nil
}

func (c *Client) CreateDisk(vm Vm, d Disk) (string, error) {
	var id string
	params := map[string]interface{}{
		"name": d.NameLabel,
		"size": d.Size,
		"sr":   d.SrId,
		"vm":   vm.Id,
	}
	err := c.Call("disk.create", params, &id)

	return id, err
}

func (c *Client) DeleteDisk(vm Vm, d Disk) error {
	var success bool
	disconnectParams := map[string]interface{}{
		"id": d.Id,
	}
	err := c.Call("vbd.disconnect", disconnectParams, &success)

	if err != nil {
		return err
	}

	deleteParams := map[string]interface{}{
		"id": d.Id,
	}
	err = c.Call("vbd.delete", deleteParams, &success)

	if err != nil {
		return err
	}

	vdiDeleteParams := map[string]interface{}{
		"id": d.VDIId,
	}
	return c.Call("vdi.delete", vdiDeleteParams, &success)
}

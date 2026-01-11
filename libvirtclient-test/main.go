package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	// Parse command-line flags
	createVM := flag.String("create", "", "Create VM with specified name")
	deleteVM := flag.String("delete", "", "Delete specified VM and its associated resources")
	statusVM := flag.String("status", "", "Check status of specified VM")
	reconcileVM := flag.String("reconcile", "", "Check specified VM status and recreate if drift detected")
	flag.Parse()

	externalMachine := LibvirtClientMachine{
		Name:             "", // to be set based on flags
		NetworkName:      "k8s",
		StoragePoolName:  "k8s",
		CPU:              1,
		Memory:           1024,
		DiskSize:         5,
		BackingImagePath: "/k8s/noble-server-cloudimg-amd64.img",
		UserData: `#cloud-config
users:
  - name: ubuntu
    ssh-authorized-keys:
      - ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCbQPpQrvr9OsdfI/+UA9j3xJa8KTPs2qvawuAFtyS4gGsXpyLov5Ilhe5m9b07/aSFhjdCBGPlzqBmuO603h+C+/tszOMUzQWQdDsjEsiIO6M//n9JYboK6PqignkrncwjC1P3h5wzj/3XFsR/93OBWu49tP5k52lt8bsHr12atpHxjQDql4CyKb7ACPxbqIc4ee+hlvW/mrqRO+Q5kZxAPfIpc1yFeLhlATIVpRX5mH9oRYG1N8C7u2/rtJAJTjevxqMgmGxpqFqeUUcaytVFOCpGy0Rg4+h3qurlYxIVPc6MInOZCy0/40JWS0xwqMVqYKUnULGfYh4KHeItv/9OqtBoOORrDywUd0r0XW5YB2nqb57JiHJSiqnC0RH+/3O6ITFCx/4LItWr8G1ogI9kOD9/3H4wO1g8WRjWxnPzVMk2sXm5eSDZfhmoDqhQv+d/WEEWRLTmXnbp5LCQZDtC2M0b8wGohVnc+xIif6qJlo5r1rVpkDeuCJzgqh2ECOS4fe5Se/AhI63+GbVFdcmLL6LLszUC9egGrLTIzOJNgd0foyt22CeXRomwoMiTjkKr1Ih5Aet+POkKcdMCZEtwGWdoPp/n+TAI7ba/r5+17I0EOkdnBnhmxxGZL4EBOK4rjg0FfD2McVeaXPm4oOGKM2WS0AnY//1KDCRXAiAfyQ== joshua@thinkpad
    sudo: ['ALL=(ALL) NOPASSWD:ALL']
    shell: /bin/bash

chpasswd:
  expire: false
  list:
  - ubuntu:password1

ssh_pwauth: true

packages:
  - iptables
  - curl

manage_etc_hosts: true
`}

	// Handle reconcile operation
	if *reconcileVM != "" {
		externalMachine.Name = *reconcileVM
		if !externalMachine.Exists() {
			fmt.Println("⚠ VM does not exist. Creating...")
			externalMachine.Create()
			return
		} else {
			if !externalMachine.IsReconciled() {
				fmt.Println("⚠ VM is not reconciled. Recreating...")
				if err := externalMachine.Destroy(); err != nil {
					log.Fatalf("failed to destroy out-of-sync VM: %v", err)
				}
				if err := externalMachine.Create(); err != nil {
					log.Fatalf("failed to recreate out-of-sync VM: %v", err)
				}
				return
			} else {
				fmt.Println("✓ No drift detected. VM matches expected configuration.")
			}
		}
		return
	}

	// Handle status check operation
	if *statusVM != "" {
		externalMachine.Name = *statusVM
		if externalMachine.IsReady() {
			fmt.Println("✓ VM is running.")
		} else {
			fmt.Println("⚠ VM is not running!")
		}
		addresses, err := externalMachine.GetIPAddresses()
		if err != nil {
			log.Fatalf("failed to get VM IP addresses: %v", err)
		}
		fmt.Printf("  IP Addresses: %v\n", addresses)
		return
	}

	// Handle delete operation
	if *deleteVM != "" {
		externalMachine.Name = *deleteVM
		if err := externalMachine.Destroy(); err != nil {
			log.Fatalf("failed to delete VM: %v", err)
		} else {
			fmt.Printf("✓ VM %s deleted successfully.\n", *deleteVM)
		}
		return
	}

	// Handle create operation
	if *createVM != "" {
		externalMachine.Name = *createVM
		if err := externalMachine.Create(); err != nil {
			log.Fatalf("failed to create VM: %v", err)
		} else {
			fmt.Printf("✓ VM %s created successfully.\n", *createVM)
		}
		return
	}

	log.Fatalln("No valid operation specified. Use create, delete, status, or reconcile operations, followed by the desired VM name.")
}

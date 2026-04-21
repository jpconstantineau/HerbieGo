package scenario

import "github.com/jpconstantineau/herbiego/internal/domain"

// Part returns one canonical part definition.
func (d Definition) Part(partID domain.PartID) (Part, bool) {
	part, ok := d.partsByID()[partID]
	return part, ok
}

// Product returns one canonical product definition.
func (d Definition) Product(productID domain.ProductID) (Product, bool) {
	product, ok := d.productsByID()[productID]
	return product, ok
}

// Workstation returns one canonical workstation definition.
func (d Definition) Workstation(workstationID domain.WorkstationID) (Workstation, bool) {
	workstation, ok := d.workstationsByID()[workstationID]
	return workstation, ok
}

// Customer returns one canonical customer definition.
func (d Definition) Customer(customerID domain.CustomerID) (CustomerMarket, bool) {
	customer, ok := d.customersByID()[customerID]
	return customer, ok
}

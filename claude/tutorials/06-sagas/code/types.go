package main

import "time"

// Travel booking request
type TravelBooking struct {
	BookingID  string     `json:"bookingId"`
	CustomerID string     `json:"customerId"`
	FlightInfo FlightInfo `json:"flightInfo"`
	HotelInfo  HotelInfo  `json:"hotelInfo"`
	CarInfo    CarInfo    `json:"carInfo"`
}

type FlightInfo struct {
	From       string    `json:"from"`
	To         string    `json:"to"`
	DepartDate time.Time `json:"departDate"`
	ReturnDate time.Time `json:"returnDate"`
	Passengers int       `json:"passengers"`
}

type HotelInfo struct {
	Location string    `json:"location"`
	CheckIn  time.Time `json:"checkIn"`
	CheckOut time.Time `json:"checkOut"`
	Guests   int       `json:"guests"`
}

type CarInfo struct {
	Location   string    `json:"location"`
	PickupDate time.Time `json:"pickupDate"`
	ReturnDate time.Time `json:"returnDate"`
	CarType    string    `json:"carType"`
}

// Confirmation responses
type FlightConfirmation struct {
	ConfirmationCode string    `json:"confirmationCode"`
	FlightNumber     string    `json:"flightNumber"`
	BookedAt         time.Time `json:"bookedAt"`
}

type HotelConfirmation struct {
	ConfirmationCode string    `json:"confirmationCode"`
	HotelName        string    `json:"hotelName"`
	BookedAt         time.Time `json:"bookedAt"`
}

type CarConfirmation struct {
	ConfirmationCode string    `json:"confirmationCode"`
	CarModel         string    `json:"carModel"`
	BookedAt         time.Time `json:"bookedAt"`
}

// Final saga result
type BookingResult struct {
	Status             string `json:"status"` // "confirmed", "failed"
	FlightConfirmation string `json:"flightConfirmation,omitempty"`
	HotelConfirmation  string `json:"hotelConfirmation,omitempty"`
	CarConfirmation    string `json:"carConfirmation,omitempty"`
	FailureReason      string `json:"failureReason,omitempty"`
}

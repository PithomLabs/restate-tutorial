package main

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
)

type TravelSaga struct{}

// Run executes the travel booking saga
func (TravelSaga) Run(
	ctx restate.WorkflowContext,
	booking TravelBooking,
) (BookingResult, error) {
	bookingID := restate.Key(ctx)

	ctx.Log().Info("Starting travel booking saga",
		"bookingId", bookingID,
		"customer", booking.CustomerID)

	// Initialize services
	flightSvc := FlightService{}
	hotelSvc := HotelService{}
	carSvc := CarService{}

	// Step 1: Reserve Flight
	ctx.Log().Info("Step 1: Reserving flight")
	flightConf, err := flightSvc.Reserve(ctx, booking.FlightInfo)
	if err != nil {
		ctx.Log().Error("Flight reservation failed", "error", err)
		return BookingResult{
			Status:        "failed",
			FailureReason: fmt.Sprintf("flight: %v", err),
		}, nil
	}
	ctx.Log().Info("Flight reserved", "code", flightConf.ConfirmationCode)

	// Step 2: Reserve Hotel
	ctx.Log().Info("Step 2: Reserving hotel")
	hotelConf, err := hotelSvc.Reserve(ctx, booking.HotelInfo)
	if err != nil {
		ctx.Log().Error("Hotel reservation failed", "error", err)

		// COMPENSATE: Cancel flight
		ctx.Log().Warn("Compensating: cancelling flight")
		flightSvc.Cancel(ctx, flightConf.ConfirmationCode)

		return BookingResult{
			Status:        "failed",
			FailureReason: fmt.Sprintf("hotel: %v", err),
		}, nil
	}
	ctx.Log().Info("Hotel reserved", "code", hotelConf.ConfirmationCode)

	// Step 3: Reserve Car
	ctx.Log().Info("Step 3: Reserving car")
	carConf, err := carSvc.Reserve(ctx, booking.CarInfo)
	if err != nil {
		ctx.Log().Error("Car reservation failed", "error", err)

		// COMPENSATE: Cancel hotel and flight
		ctx.Log().Warn("Compensating: cancelling hotel and flight")
		hotelSvc.Cancel(ctx, hotelConf.ConfirmationCode)
		flightSvc.Cancel(ctx, flightConf.ConfirmationCode)

		return BookingResult{
			Status:        "failed",
			FailureReason: fmt.Sprintf("car: %v", err),
		}, nil
	}
	ctx.Log().Info("Car reserved", "code", carConf.ConfirmationCode)

	// All steps succeeded!
	ctx.Log().Info("Travel booking completed successfully",
		"flight", flightConf.ConfirmationCode,
		"hotel", hotelConf.ConfirmationCode,
		"car", carConf.ConfirmationCode)

	return BookingResult{
		Status:             "confirmed",
		FlightConfirmation: flightConf.ConfirmationCode,
		HotelConfirmation:  hotelConf.ConfirmationCode,
		CarConfirmation:    carConf.ConfirmationCode,
	}, nil
}

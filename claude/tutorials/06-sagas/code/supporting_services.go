package main

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
)

// FlightService simulates airline booking API
type FlightService struct{}

func (FlightService) Reserve(
	ctx restate.Context,
	info FlightInfo,
) (FlightConfirmation, error) {
	// Simulate API call
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Calling flight booking API",
			"from", info.From,
			"to", info.To,
			"passengers", info.Passengers)

		time.Sleep(100 * time.Millisecond) // Simulate latency

		// 10% failure rate
		if restate.Rand(ctx).Float64() < 0.1 {
			return false, fmt.Errorf("flight unavailable")
		}

		return true, nil
	})

	if err != nil {
		return FlightConfirmation{}, err
	}

	return FlightConfirmation{
		ConfirmationCode: fmt.Sprintf("FL-%s", restate.UUID(ctx).String()[:8]),
		FlightNumber:     "AA123",
		BookedAt:         time.Now(),
	}, nil
}

func (FlightService) Cancel(
	ctx restate.Context,
	confirmationCode string,
) error {
	ctx.Log().Info("Cancelling flight", "code", confirmationCode)

	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		// Idempotent cancellation
		ctx.Log().Info("Flight cancelled successfully")
		return true, nil
	})

	return err
}

// HotelService simulates hotel booking API
type HotelService struct{}

func (HotelService) Reserve(
	ctx restate.Context,
	info HotelInfo,
) (HotelConfirmation, error) {
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Calling hotel booking API",
			"location", info.Location,
			"guests", info.Guests)

		time.Sleep(100 * time.Millisecond)

		// 10% failure rate
		if restate.Rand(ctx).Float64() < 0.1 {
			return false, fmt.Errorf("hotel unavailable")
		}

		return true, nil
	})

	if err != nil {
		return HotelConfirmation{}, err
	}

	return HotelConfirmation{
		ConfirmationCode: fmt.Sprintf("HT-%s", restate.UUID(ctx).String()[:8]),
		HotelName:        "Grand Hotel",
		BookedAt:         time.Now(),
	}, nil
}

func (HotelService) Cancel(
	ctx restate.Context,
	confirmationCode string,
) error {
	ctx.Log().Info("Cancelling hotel", "code", confirmationCode)

	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Hotel cancelled successfully")
		return true, nil
	})

	return err
}

// CarService simulates car rental API
type CarService struct{}

func (CarService) Reserve(
	ctx restate.Context,
	info CarInfo,
) (CarConfirmation, error) {
	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Calling car rental API",
			"location", info.Location,
			"type", info.CarType)

		time.Sleep(100 * time.Millisecond)

		// 10% failure rate
		if restate.Rand(ctx).Float64() < 0.1 {
			return false, fmt.Errorf("car unavailable")
		}

		return true, nil
	})

	if err != nil {
		return CarConfirmation{}, err
	}

	return CarConfirmation{
		ConfirmationCode: fmt.Sprintf("CR-%s", restate.UUID(ctx).String()[:8]),
		CarModel:         info.CarType,
		BookedAt:         time.Now(),
	}, nil
}

func (CarService) Cancel(
	ctx restate.Context,
	confirmationCode string,
) error {
	ctx.Log().Info("Cancelling car rental", "code", confirmationCode)

	_, err := restate.Run(ctx, func(ctx restate.RunContext) (bool, error) {
		ctx.Log().Info("Car rental cancelled successfully")
		return true, nil
	})

	return err
}

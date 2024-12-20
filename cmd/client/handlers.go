package main

import (
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerArmyMoves(gs *gamelogic.GameState, ch *amqp.Channel) func(mv gamelogic.ArmyMove) pubsub.AckType {
	return func(mv gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(mv)
		switch outcome {
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gs.GetUsername()),
				gamelogic.RecognitionOfWar{
					Attacker: mv.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Printf("could not publish war recognition: %v\n", err)
				return pubsub.NackRequeue
			}

			return pubsub.Ack
		}

		fmt.Println("unexpected outcome")
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(rw)

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeYouWon, gamelogic.WarOutcomeOpponentWon:
			err := pubsub.PublishGob(
				ch,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.GameLogSlug, rw.Attacker.Username),
				routing.GameLog{
					CurrentTime: time.Now(),
					Username:    gs.GetUsername(),
					Message:     fmt.Sprintf("%s won a war against %s", winner, loser),
				},
			)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			err := pubsub.PublishGob(
				ch,
				routing.ExchangePerilTopic,
				fmt.Sprintf("%s.%s", routing.GameLogSlug, rw.Attacker.Username),
				routing.GameLog{
					CurrentTime: time.Now(),
					Username:    gs.GetPlayerSnap().Username,
					Message:     fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser),
				},
			)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}

		fmt.Println("unexpected outcome")
		return pubsub.NackDiscard
	}
}

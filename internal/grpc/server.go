package grpc

import (
	"HellgameProject/internal/engine"
	"HellgameProject/internal/grpc/pb"
	"context"
	"encoding/json"
)

type Server struct {
	sim engine.GameEngine
	pb.UnimplementedGameEngineServer
}

// Создаём сервер через конструктор
func NewServer(sim engine.GameEngine) *Server {
	return &Server{sim: sim}
}

// Реализуем методы gRPC сервера

func (s *Server) Simulate(ctx context.Context, req *pb.SimulateRequest) (*pb.SimulateResponse, error) {
	if req.Ticks <= 0 {
		req.Ticks = 1
	}
	delta := s.sim.Simulate(req.Ticks)
	return &pb.SimulateResponse{
		CurrentTick:    delta.GlobalTick,
		TicksSimulated: delta.TicksSimulated,
		Success:        true,
	}, nil
}

func (s *Server) GetWorldState(ctx context.Context, req *pb.GetWorldStateRequest) (*pb.GetWorldStateResponse, error) {
	state := s.sim.GetWorldState()
	pbFactions := make(map[string]*pb.FactionState)
	for id, faction := range state.Factions {
		pbFactions[id] = &pb.FactionState{
			Id:              faction.ID,
			Name:            faction.Name,
			Power:           faction.Power,
			Territory:       faction.Territory,
			DomainsHeld:     faction.DomainsHeld,
			Attitude:        faction.Attitude,
			MilitaryForce:   faction.MilitaryForce,
			LastActionTime:  faction.LastActionTime,
			WealthIndex:     faction.WealthIndex,
			TotalPopulation: int64(faction.TotalPopulation),
		}
	}

	pbDomains := make(map[string]*pb.DomainState)
	for id, domain := range state.Domains {
		pbDomains[id] = &pb.DomainState{
			Id:              domain.ID,
			Name:            domain.Name,
			Stability:       domain.Stability,
			ControlledBy:    domain.ControlledBy,
			DangerLevel:     domain.DangerLevel,
			Population:      int32(domain.Population),
			Mood:            domain.Mood,
			Events:          domain.Events,
			Influence:       domain.Influence,
			AdjacentDomains: domain.AdjacentDomains,
			Resources:       domain.Resources,
		}
	}

	pbWars := make(map[string]*pb.WarState)
	for id, war := range state.Wars {
		pbWars[id] = &pb.WarState{
			Id:                      war.ID,
			AttackerId:              war.AttackerID,
			DefenderId:              war.DefenderID,
			StartTick:               war.StartTick,
			LastUpdateTick:          war.LastUpdateTick,
			TicksDuration:           war.TicksDuration,
			FrozenFactionsDenseties: war.FrozenFactionsDenseties,
			AttackerCommittedForce:  war.AttackerCommittedForce,
			DefenderCommittedForce:  war.DefenderCommittedForce,
			AttackerCurrentForce:    war.AttackerCurrentForce,
			DefenderCurrentForce:    war.DefenderCurrentForce,
			Momentum:                war.Momentum,
			AttackerMorale:          war.AttackerMorale,
			DefenderMorale:          war.DefenderMorale,
			IsOver:                  war.IsOver,
			WinnersId:               war.WinnersID,
			LosersId:                war.LosersID,
		}
	}

	return &pb.GetWorldStateResponse{
		Time:     state.Time,
		Factions: pbFactions,
		Domains:  pbDomains,
		Wars:     pbWars,
	}, nil
}

func (s *Server) GetFactions(ctx context.Context, req *pb.GetFactionsRequest) (*pb.GetFactionsResponse, error) {
	state := s.sim.GetWorldState()
	pbFactions := make(map[string]*pb.FactionState)
	for id, faction := range state.Factions {
		pbFactions[id] = &pb.FactionState{
			Id:              faction.ID,
			Name:            faction.Name,
			Power:           faction.Power,
			Territory:       faction.Territory,
			DomainsHeld:     faction.DomainsHeld,
			Attitude:        faction.Attitude,
			MilitaryForce:   faction.MilitaryForce,
			LastActionTime:  faction.LastActionTime,
			WealthIndex:     faction.WealthIndex,
			TotalPopulation: int64(faction.TotalPopulation),
		}
	}

	return &pb.GetFactionsResponse{
		Factions: pbFactions,
	}, nil
}

func (s *Server) GetDomains(ctx context.Context, req *pb.GetDomainsRequest) (*pb.GetDomainsResponse, error) {
	state := s.sim.GetWorldState()
	pbDomains := make(map[string]*pb.DomainState)
	for id, domain := range state.Domains {
		pbDomains[id] = &pb.DomainState{
			Id:              domain.ID,
			Name:            domain.Name,
			Stability:       domain.Stability,
			ControlledBy:    domain.ControlledBy,
			DangerLevel:     domain.DangerLevel,
			Population:      int32(domain.Population),
			Mood:            domain.Mood,
			Events:          domain.Events,
			Influence:       domain.Influence,
			AdjacentDomains: domain.AdjacentDomains,
			Resources:       domain.Resources,
		}
	}

	return &pb.GetDomainsResponse{
		Domains: pbDomains,
	}, nil
}

func (s *Server) GetEvents(ctx context.Context, req *pb.GetEventsRequest) (*pb.GetEventsResponse, error) {
	// 1. Берем лимит из запроса (с защитой от нуля)
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}

	// 2. Получаем сырые события из ядра
	coreEvents := s.sim.GetEvents(limit)

	// 3. Создаем слайс для gRPC событий
	pbEvents := make([]*pb.GameEvent, 0, len(coreEvents))

	// 4. Перекладываем каждое событие
	for _, ce := range coreEvents {
		// Базовые поля
		grpcEvent := &pb.GameEvent{
			Type:      ce.Type,
			Tick:      ce.Tick,
			EventKind: string(ce.EventKind),
		}

		// РАСПАКОВКА ИНТЕРФЕЙСА:
		// Смотрим, какой конкретный тип лежит внутри EventData
		switch data := ce.EventData.(type) {

		case *engine.WarStartData: // или engine.WarStartData (зависит от того, как вы храните)
			grpcEvent.EventData = &pb.GameEvent_WarStarted{ // Это спец. обертка, сгенерированная protoc
				WarStarted: &pb.WarStartedData{
					Attacker:              data.Attacker,
					Defender:              data.Defender,
					Domain:                data.Domain,
					DomainId:              data.DomainID,
					Reason:                data.Reason,
					ActualDefenderForce:   data.ActualDefenderForce,
					EstimateDefenderForce: data.EstimateDefenderForce,
					AttackerCommited:      data.AttackerCommitted,
					DefenderCommited:      data.DefenderCommitted,
					Ration:                data.Ratio,
					MinStrengthRatio:      data.MinStrengthRatio,
				},
			}

		case *engine.WarEndedData:
			grpcEvent.EventData = &pb.GameEvent_WarEnded{
				WarEnded: &pb.WarEndedData{
					Attacker:            data.Attacker,
					Defender:            data.Defender,
					Domain:              data.Domain,
					Reason:              data.Reason,
					WinnerId:            data.WinnerID,
					LosersId:            data.LoserID,
					AttackerLossesPct:   data.AttackerLossesPct,
					DefenderLossesPct:   data.DefenderLossesPct,
					AttackerMorale:      data.AttackerMorale,
					DefenderMorale:      data.DefenderMorale,
					AttackerForceRemain: data.AttackerForceRemain,
					DefenderForceRemain: data.DefenderForceRemain,
				},
			}
		case *engine.WarUpdateData:
			grpcEvent.EventData = &pb.GameEvent_WarUpdate{
				WarUpdate: &pb.WarUpdateData{
					Attacker:          data.Attacker,
					Defender:          data.Defender,
					Domain:            data.Domain,
					Momentum:          data.Momentum,
					AttackerMorale:    data.AttackerMorale,
					DefenderMorale:    data.DefenderMorale,
					AttackerForce:     data.AttackerForce,
					DefenderForce:     data.DefenderForce,
					AttackerLossesPct: data.AttackerLossesPct,
					DefenderLossesPct: data.DefenderLossesPct,
					ForceRatio:        data.ForceRatio,
				},
			}

		case *engine.WarAbortedData:
			grpcEvent.EventData = &pb.GameEvent_WarAborted{
				WarAborted: &pb.WarAbortedData{
					Attacker:              data.Attacker,
					Defender:              data.Defender,
					Domain:                data.Domain,
					DomainId:              data.DomainID,
					Reason:                data.Reason,
					ActualDefenderForce:   data.ActualDefenderForce,
					EstimateDefenderForce: data.EstimateDefenderForce,
					AttackerCommited:      data.AttackerCommitted,
					DefenderCommited:      data.DefenderCommitted,
					Ratio:                 data.Ratio,
					MinStrengthRatio:      data.MinStrengthRatio,
				},
			}

		case *engine.GenericEventData:
			// Как мы обсуждали, тут можно просто в JSON сбросить внутреннюю мапу
			jsonBytes, _ := json.Marshal(data.EventData)
			grpcEvent.EventData = &pb.GameEvent_GenericEvent{
				GenericEvent: &pb.GenericEventData{
					EventKind:   string(data.EventKind),
					PayloadJson: string(jsonBytes),
				},
			}
		}

		pbEvents = append(pbEvents, grpcEvent)
	}

	// 5. Отдаем результат
	return &pb.GetEventsResponse{
		Count:  int32(len(pbEvents)),
		Events: pbEvents,
	}, nil
}

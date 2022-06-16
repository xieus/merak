package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	pb "github.com/futurewei-cloud/merak/api/proto/v1/merak"
	"github.com/futurewei-cloud/merak/services/scenario-manager/database"
	"github.com/futurewei-cloud/merak/services/scenario-manager/entities"
	"github.com/futurewei-cloud/merak/services/scenario-manager/handler"
	"github.com/futurewei-cloud/merak/services/scenario-manager/utils"
	"github.com/gofiber/fiber/v2"
)

//Function for doing some actions for a scenario
//@Summary Do something on a scenario
//@Description Take an action on a scenario
//@Tags scenario
//@Accept json
//@Product json
//@Param scenario body entities.ScenarioAction true "ScenarioAction"
//@Success 200 {object} entities.ScenarioAction "scenario action data with success message"
//@Failure 500 {object} nil "scenario action null with failure message"
//@Router /api/scenarios/action [post]
func ScenarioActoins(c *fiber.Ctx) error {
	var scenario_action entities.ScenarioAction

	if err := c.BodyParser(&scenario_action); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
	}

	var scenario entities.Scenario
	if err := database.FindEntity(scenario_action.ScenarioId, utils.KEY_PREFIX_SCENARIO, &scenario); err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.ReturnResponseMessage("FAILED", "Scenario not found!", nil))
	}

	// if scenario.Status != entities.STATUS_NONE {
	// 	return c.Status(http.StatusNotAcceptable).JSON(utils.ReturnResponseMessage("FAILED", "Scenario is not available now!", nil))
	// }

	if err := checkRelatedEntities(&scenario); err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
	}

	scenario.Status = entities.STATUS_DEPLOYING
	database.Set(utils.KEY_PREFIX_SCENARIO+scenario.Id, &scenario)

	var returnTopo *pb.ReturnTopologyMessage
	var returnNetwork *pb.ReturnNetworkMessage
	var returnCompute *pb.ReturnMessage
	for _, sa := range scenario_action.Services {
		if strings.ToLower(sa.ServiceName) == "topology" {
			if err := handler.TopologyHandler(&scenario, returnTopo); err != nil {
				return c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
			}
		}
		if strings.ToLower(sa.ServiceName) == "network" {
			if err := handler.NetworkHandler(&scenario, returnTopo, returnNetwork); err != nil {
				return c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
			}
		}
		if strings.ToLower(sa.ServiceName) == "compute" {
			if err := handler.ComputeHanlder(&scenario, returnTopo, returnNetwork, returnCompute); err != nil {
				return c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
			}
		}
	}

	scenario.Status = entities.STATUS_DEPLOYING
	scenario.UpdatedAt = time.Now()

	database.Set(utils.KEY_PREFIX_SCENARIO+scenario.Id, &scenario)

	return c.Status(http.StatusOK).JSON(utils.ReturnResponseMessage("OK", "Scenario is deploying.", scenario))
}

//Function for creating a scenario
//@Summary Insert a scenario to database
//@Description Create a scenario
//@Tags scenario
//@Accept json
//@Product json
//@Param scenario body entities.Scenario true "Scenario"
//@Success 200 {object} entities.Scenario "scenario data with success message"
//@Failure 500 {object} nil "scenario null with failure message"
//@Router /api/scenarios [post]
func CreateScenario(c *fiber.Ctx) error {
	var scenario entities.Scenario

	if err := c.BodyParser(&scenario); err != nil {
		return c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
	}

	var id = utils.GenUUID()
	scenario.Id = id

	if err := checkRelatedEntities(&scenario); err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
	}

	scenario.Status = entities.STATUS_NONE
	scenario.CreatedAt = time.Now()
	scenario.UpdatedAt = time.Now()

	database.Set(utils.KEY_PREFIX_SCENARIO+id, &scenario)

	return c.Status(http.StatusOK).JSON(utils.ReturnResponseMessage("OK", "Scenario Has been created successfully.", scenario))
}

//Function for retriving all scenarios
//@Summary Get all scenario from database
//@Description Get all scenario
//@Tags scenario
//@Accept json
//@Product json
//@Success 200 {object} []entities.Scenario "array of scenario with success message"
//@Failure 404 {object} nil "null scenario data with error message"
//@Router /api/senarios [get]
func GetScenarios(c *fiber.Ctx) error {
	var values map[string]string

	values, err := database.GetAllValuesWithKeyPrefix(utils.KEY_PREFIX_SCENARIO)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
	}

	if len(values) < 1 {
		return c.Status(http.StatusNotFound).JSON(utils.ReturnResponseMessage("FAILED", errors.New("scenario not present").Error(), nil))
	}

	var responseScenarios []entities.Scenario

	for _, value := range values {
		var scenario entities.Scenario

		_ = json.Unmarshal([]byte(value), &scenario)
		responseScenarios = append(responseScenarios, scenario)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "OK", "message": "OK", "data": responseScenarios})
}

//Function for retriving a scenario
//@Summary Get a scenario from database
//@Description Get a scenario
//@Tags scenario
//@Accept json
//@Product json
//@Param id path string true "ScenarioId"
//@Success 200 {object} entities.Scenario "scenario data with success message"
//@Failure 404 {object} nil "scenario data with null and error message"
//@Router /api/senarios/{id} [get]
func GetScenario(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", "Scenario id is missing!", nil))
	}

	var scenario entities.Scenario
	if err := database.FindEntity(id, utils.KEY_PREFIX_SCENARIO, &scenario); err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.ReturnResponseMessage("FAILED", "Scenario not found!", nil))
	}

	return c.Status(http.StatusOK).JSON(utils.ReturnResponseMessage("OK", "OK", scenario))
}

//Function for updating a scenario
//@Summary Update a scenario to database
//@Description Update a scenario
//@Tags scenario
//@Accept json
//@Product json
//@Param id path string true "ScenarioId"
//@Param scenario body string true "Scenario"
//@Success 200 {object} entities.Scenario "scenario data with success message"
//@Failure 500 {object} nil "scenario null with failure message"
//@Router /api/senarios/{id} [put]
func UpdateScenario(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", "Scenario id is missing!", nil))
	}

	var scenario entities.Scenario
	if err := database.FindEntity(id, utils.KEY_PREFIX_SCENARIO, &scenario); err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.ReturnResponseMessage("FAILED", "Scenario not found!", nil))
	}

	var updateScenario entities.Scenario
	if err := c.BodyParser(&updateScenario); err != nil {
		c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
	}

	scenario.Name = updateScenario.Name
	scenario.ProjectId = updateScenario.ProjectId
	scenario.ServiceConfId = updateScenario.ServiceConfId
	scenario.TopologyId = updateScenario.TopologyId
	scenario.NetworkConfId = updateScenario.NetworkConfId
	scenario.ComputeConfId = updateScenario.ComputeConfId
	scenario.TestConfId = updateScenario.TestConfId
	scenario.UpdatedAt = time.Now()

	if err := checkRelatedEntities(&scenario); err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
	}

	database.Set(utils.KEY_PREFIX_SCENARIO+id, &scenario)

	return c.Status(http.StatusOK).JSON(utils.ReturnResponseMessage("OK", "OK", scenario))
}

// Function for delete a scenario
// @Summary Delete a scenario from database
// @Description Delete a scenario
// @Tags scenario
// @Accept json
// @Product json
// @Param id path string true "ScenarioId"
// @Success 200 {object} entities.Scenario "scenario data with success message"
// @Failure 404 {object} nil "scenario data with null and error message"
// @Router /api/senarios/{id} [delete]
func DeleteScenario(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(utils.ReturnResponseMessage("FAILED", "Scenario id is missing!", entities.Scenario{}))
	}

	var scenario entities.Scenario
	if err := database.FindEntity(id, utils.KEY_PREFIX_SCENARIO, &scenario); err != nil {
		return c.Status(http.StatusNotFound).JSON(utils.ReturnResponseMessage("FAILED", "Scenario not found!", nil))
	}

	if err := database.Del(utils.KEY_PREFIX_SCENARIO + id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(utils.ReturnResponseMessage("FAILED", err.Error(), nil))
	}

	return c.Status(http.StatusOK).JSON(utils.ReturnResponseMessage("OK", "Scenario has been deleted!", nil))
}

func checkRelatedEntities(scenario *entities.Scenario) error {
	var topology entities.TopologyConfig
	if err := database.FindEntity(scenario.TopologyId, utils.KEY_PREFIX_TOPOLOGY, &topology); err != nil {
		return errors.New("topology not found")
	}

	var service entities.ServiceConfig
	if err := database.FindEntity(scenario.ServiceConfId, utils.KEY_PREFIX_SERVICE, &service); err != nil {
		return errors.New("service config not found")
	}

	var network entities.NetworkConfig
	if err := database.FindEntity(scenario.NetworkConfId, utils.KEY_PREFIX_NETWORK, &network); err != nil {
		return errors.New("network config not found")
	}

	var compute entities.ComputeConfig
	if err := database.FindEntity(scenario.ComputeConfId, utils.KEY_PREFIX_COMPUTE, &compute); err != nil {
		return errors.New("compute config not found")
	}

	var test entities.TestConfig
	if err := database.FindEntity(scenario.TestConfId, utils.KEY_PREFIX_TEST, &test); err != nil {
		return errors.New("test config not found")
	}

	return nil
}
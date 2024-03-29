package ginger

import (
	"github.com/gin-gonic/gin"
	"github.com/kulichak/dl"
	"github.com/kulichak/logic"
	"github.com/kulichak/models"
	"github.com/kulichak/models/errors"
	"strconv"
)

type IController interface {
	GetRequestSample() models.IRequest
	GetRoutes() []BaseControllerRoute

	GetHandler(group *RouterGroup, routeHandler RouteHandler) gin.HandlerFunc

	post(request models.IRequest) (result interface{})
	get(request models.IRequest) (result interface{})
	put(request models.IRequest) (result interface{})

	Post(request models.IRequest) (result interface{})
	Get(request models.IRequest) (result interface{})
	Put(request models.IRequest) (result interface{})
	Delete(request models.IRequest) (result interface{})
}

type HandlerFunc func(request models.IRequest) (result interface{})
type RouteHandler struct {
	Type     int
	Handler  HandlerFunc
	CallBack func(request models.IRequest, extra interface{})
}

type BaseControllerRoute struct {
	Method   string
	Handlers []RouteHandler
}

type BaseController struct {
	IController

	Controller   IController
	Routes       []BaseControllerRoute
	LogicHandler logic.IBaseLogicHandler
}

func (c *BaseController) Init(controller IController, logicHandler logic.IBaseLogicHandler, dbHandler dl.IBaseDbHandler) {
	c.Controller = controller
	c.LogicHandler = logicHandler
	if c.LogicHandler != nil {
		c.LogicHandler.Init(logicHandler, dbHandler)
	}
}

func (c *BaseController) GetRequestSample() models.IRequest {
	return &models.Request{}
}

func (c *BaseController) AddRoute(method string, handlers ...HandlerFunc) {
	routeHandlers := make([]RouteHandler, 0)
	for _, handler := range handlers {
		routeHandlers = append(routeHandlers, RouteHandler{
			Handler: handler,
		})
	}
	c.Routes = append(c.Routes, BaseControllerRoute{
		Method:   method,
		Handlers: routeHandlers,
	})
}

func (c *BaseController) GetHandler(group *RouterGroup, routeHandler RouteHandler) gin.HandlerFunc {
	return func(context *gin.Context) {
		var request models.IRequest
		if req, ok := context.Keys["request"]; ok {
			request = req.(models.IRequest)
		}
		var result interface{}
		if routeHandler.Type == -1 {
			req, err := c.NewRequest(context)
			if c.HandleErrorNoResult(req, err) {
				context.Abort()
				return
			}
			if context.Keys == nil {
				context.Keys = map[string]interface{}{}
			}
			context.Keys["request"] = req
		} else {
			if routeHandler.Handler != nil {
				result := routeHandler.Handler(request)
				if result != nil {
					if err, ok := result.(error); ok {
						if c.HandleErrorNoResult(request, err) {
							return
						}
					}
				}
			}
		}
		if routeHandler.CallBack != nil {
			routeHandler.CallBack(request, result)
		}
	}
}

func (c *BaseController) AddRouteWithCallback(method string, handlers ...RouteHandler) {
	c.Routes = append(c.Routes, BaseControllerRoute{
		Method:   method,
		Handlers: handlers,
	})
}

func (c *BaseController) GetRoutes() []BaseControllerRoute {
	return c.Routes
}

func (c *BaseController) handleError(err error) (*int, error) {
	if err != nil {
		status := 400
		message := err.Error()
		if e, ok := err.(errors.Error); ok {
			status = e.Status
		}
		if status == 0 {
			status = 400
		}
		return &status, &errors.Error{
			Status:  status,
			Message: message,
		}
	}
	return nil, nil
}

func (c *BaseController) HandleErrorNoResult(request models.IRequest, err error) (handled bool) {
	if err != nil {
		status, e := c.handleError(err)
		if status != nil && e != nil {
			req := request.GetBaseRequest()
			req.Context.JSON(*status, errors.Error{
				Message: e.Error(),
			})
			return true
		}
	}
	return false
}

func (c *BaseController) HandleError(request models.IRequest, result interface{}, err error) (handled bool) {
	req := request.GetBaseRequest()
	if err != nil {
		status, e := c.handleError(err)
		if status != nil && e != nil {
			req.Context.JSON(*status, errors.Error{
				Message: e.Error(),
			})
			return true
		}
	} else if result == nil {
		req.Context.JSON(404, result)
		return true
	}
	return false
}

func (c *BaseController) handleFilters(request models.IRequest) {
	context := request.GetContext()
	req := request.GetBaseRequest()
	var v models.Filters = GetQueryFilters(context)
	if v != nil {
		req.Filters = &v
	}
	context.Set("filters", v)
}

func (c *BaseController) handlePagination(request models.IRequest) {
	context := request.GetContext()
	req := request.GetBaseRequest()

	var v = GetSortFields(context)
	if v != nil {
		req.Sort = &v
	}

	if q, ok := context.GetQuery("page"); ok {
		page, _ := strconv.ParseUint(q, 10, 32)
		req.Page = page
	}
	if q, ok := context.GetQuery("per_page"); ok {
		perPage, _ := strconv.ParseUint(q, 10, 32)
		req.PerPage = perPage
	}
}

func (c *BaseController) handleFields(request models.IRequest) {
	context := request.GetContext()
	req := request.GetBaseRequest()
	var f models.Fields = GetFetchFields(context, nil)
	if f != nil {
		req.Fields = &f
	}
	context.Set("fields", f)
}

func (c *BaseController) post(request models.IRequest) (result interface{}) {
	return
}

func (c *BaseController) get(request models.IRequest) (result interface{}) {
	c.handleFields(request)
	c.handleFilters(request)
	return
}

func (c *BaseController) put(request models.IRequest) (result interface{}) {
	return
}

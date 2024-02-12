package controller

import (
	"advertisement-api/internal/dto"
	"advertisement-api/internal/model"
	"advertisement-api/internal/repository"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type AdController struct {
    adRepository repository.AdRepository
    redis       *redis.Client
}

func NewAdController(adRepo repository.AdRepository, redis *redis.Client) *AdController {
    return &AdController{
        adRepository: adRepo,
        redis:       redis,
    }
}

// GetAd godoc
// @Summary Get advertisements
// @Description get advertisements by params and conditions
// @Tags advertisements
// @Accept json
// @Produce json
// @@Param offset query int true "Offset"
// @Param limit query int true "Limit<1~100,default=5>"
// @Param age query int false "Age <1~100>"
// @Param gender query string false "Gender <enum:M、F>"
// @Param country query string false "Country <enum:TW、JP 等符合 https://zh.wikipedia.org/wiki/ISO_3166-1 >"
// @Param platform query string false "Platform <enum:android, ios, web>"
// @Success 200 {object} dto.AdGetResponse "success"
// @Failure 400 {object} gin.H "{"error": "params error"}"
// @Failure 500 {object} gin.H "{"error": "server error"}"
// @Router /ad [get]
func(a *AdController) GetAd(c *gin.Context) {
	var adReq dto.AdGetRequest
    err := c.ShouldBind(&adReq)
	if err != nil {
        fmt.Println("err")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
    redisKey := adReq.GetParams()

    val, err := a.redis.Get(c, redisKey).Result()
    if err == redis.Nil {
        ads, err := a.adRepository.GetActiveAdvertisements(time.Now(), adReq)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        if len(ads) > 0 {
            adsJson, err := json.Marshal(ads)
            if err == nil {
                a.redis.Set(c, redisKey, adsJson, 1*time.Minute)
            }
        }

        c.JSON(http.StatusOK, ads)
    } else if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis error"})
    } else {
        var ads []dto.AdGetResponse
        err := json.Unmarshal([]byte(val), &ads)
        if err != nil {
            fmt.Println(err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unmarshalling data"})
            return
        }
        c.JSON(http.StatusOK, ads)
    }
}

// CreateAd godoc
// @Summary Create advertisement
// @Description create a new advertisement
// @Tags advertisements
// @Accept json
// @Produce json
// @Param adCreationRequest body dto.AdCreationRequest true "Create Advertisement"
// @Success 200 {object} gin.H "{"message": "success"}"
// @Failure 400 {object} gin.H "{"error": "params error"}"
// @Failure 500 {object} gin.H "{"error": "server error"}"
// @Router /ad [post]
func(a *AdController) CreateAd(c *gin.Context) {
    var adCreate dto.AdCreationRequest
    err := c.ShouldBindJSON(&adCreate)
    if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

    ad := model.Advertisement{
        Title:     adCreate.Title,
        StartAt:   adCreate.StartAt,
        EndAt:     adCreate.EndAt,
        AgeStart:  adCreate.Conditions.AgeStart,
        AgeEnd:    adCreate.Conditions.AgeEnd,
        // 避免 nil pointer dereference
        Gender:    assignConditionValue(adCreate.Conditions.Gender),  
        Country:   assignConditionValue(adCreate.Conditions.Country),
        Platform:  assignConditionValue(adCreate.Conditions.Platform),
    }
    if err := a.adRepository.CreateAdvertisement(&ad); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func assignConditionValue(condition *[]string) []string {
    if condition != nil {
        return *condition
    }
    return []string{}
}
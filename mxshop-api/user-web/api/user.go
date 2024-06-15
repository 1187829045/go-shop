package api

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"mxshop-api/user-web/forms"
	"mxshop-api/user-web/global"
	"mxshop-api/user-web/global/reponse"
	"mxshop-api/user-web/middlewares"
	"mxshop-api/user-web/models"
	"mxshop-api/user-web/proto"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// 去除结构体字段名中的前缀
func removeTopStruct(fileds map[string]string) map[string]string {
	rsp := map[string]string{}
	for field, err := range fileds {
		rsp[field[strings.Index(field, ".")+1:]] = err
	}
	return rsp
}

// HandleGrpcErrorToHttp 将 gRPC 错误转换为 HTTP 响应
// err: gRPC 调用返回的错误
// c: gin 的上下文，用于生成 HTTP 响应
func HandleGrpcErrorToHttp(err error, c *gin.Context) {
	if err != nil {
		// 将错误转换为 gRPC 状态
		if e, ok := status.FromError(err); ok {
			// 根据错误代码返回对应的 HTTP 状态码和消息
			switch e.Code() {
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"msg": e.Message(), // 返回未找到的错误信息
				})
			case codes.Internal:
				c.JSON(http.StatusInternalServerError, gin.H{
					"msg": "内部错误", // 返回内部错误信息
				})
			case codes.InvalidArgument:
				c.JSON(http.StatusBadRequest, gin.H{
					"msg": "参数不可用", // 返回参数错误信息
				})
			case codes.Unavailable:
				c.JSON(http.StatusInternalServerError, gin.H{
					"msg": "用户服务不可用", // 返回用户服务不可用错误信息
				})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{
					"msg": "其他错误" + e.Message(), // 返回其他错误信息
				})
			}
			return
		}
	}
}

//处理 Gin 框架中请求参数绑定和验证过程中可能发生的错误，并向客户端返回适当的错误信息

func HandleValidatorError(c *gin.Context, err error) {
	//validator 是一个常用的结构体字段验证库，用于验证结构体中字段的值是否符合特定的规则或约束条件。
	//它能够有效地帮助开发者在处理用户输入、表单提交等场景中，对数据进行验证，以确保数据的合法性和完整性。
	//alidator.ValidationErrors 并不是一个函数，而是一个类型。它用于表示验证失败时返回的详细错误信息，通常在结构体字段验证失败时返回。
	// 如果绑定或验证失败，处理错误
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"msg": err.Error(),
		})
		return
	}
	// 返回验证错误信息
	c.JSON(http.StatusBadRequest, gin.H{
		"error": removeTopStruct(errs.Translate(global.Trans)),
	})
	return
}

// GetUserList 获取用户列表
// ctx: gin 的上下文，用于处理 HTTP 请求和响应
func GetUserList(ctx *gin.Context) {

	// 拨号连接用户 gRPC 服务器 跨域的问题-后端解决
	userConn, err := grpc.Dial(fmt.Sprintf("%s:%d", global.ServerConfig.UserSrvInfo.Host,
		global.ServerConfig.UserSrvInfo.Port), grpc.WithInsecure())
	if err != nil {
		zap.S().Errorw("[GetUserList]连接【用户服务失败】",
			"msg", err.Error()) // 记录连接用户服务失败的错误信息
	}
	claims, _ := ctx.Get("claims")
	currentUser := claims.(*models.CustomClaims)
	zap.S().Infof("访问用户：%d", currentUser.ID)
	// 生成 gRPC 的客户端并调用接口
	userSrvClient := proto.NewUserClient(userConn)

	// 请求用户列表的参数
	// context.Background(): 上下文，用于控制 gRPC 调用的生命周期
	// &proto.PageInfo{Pn: 0, PSize: 0}: 分页信息，Pn 为页码，PSize 为每页大小
	//Pn 和 PSize 为 0，可能表示获取所有数据，
	pn := ctx.DefaultQuery("pn", "0")
	pnInt, _ := strconv.Atoi(pn)
	pSize := ctx.DefaultQuery("pSize", "10")
	pSizeInt, _ := strconv.Atoi(pSize)

	rsp, err := userSrvClient.GetUserList(context.Background(), &proto.PageInfo{
		Pn:    uint32(pnInt),    // 页码
		PSize: uint32(pSizeInt), // 每页大小
	})
	if err != nil {
		zap.S().Errorw("[GetUserList]查询【用户列表】失败") // 记录查询用户列表失败的错误信息
		HandleGrpcErrorToHttp(err, ctx)           // 处理 gRPC 错误
		return
	}

	zap.S().Debug("获取用户列表页")         // 记录获取用户列表页的调试信息
	result := make([]interface{}, 0) // 创建一个空的结果列表

	// 遍历用户列表并构造结果
	// rsp.Data: 用户列表数据
	for _, value := range rsp.Data {
		//data := make(map[string]interface{}) // 创建一个空的 map 用于存储用户数据
		user := reponse.UserResponse{
			Id:       value.Id,
			NickName: value.NickName,
			//Birthday: time.Time(time.Unix(int64(value.BirthDay), 0)),
			Birthday: reponse.JsonTime(time.Unix(int64(value.BirthDay), 0)),
			Gender:   value.Gender,
			Mobile:   value.Mobile,
		}
		result = append(result, user) // 将用户数据添加到结果列表中
	}

	// 返回 JSON 格式的结果
	// http.StatusOK: HTTP 状态码 200 表示成功
	// result: 返回的用户列表数据
	ctx.JSON(http.StatusOK, result)
}
func PassWordLogin(c *gin.Context) {
	//表单验证
	//表单验证是一种在用户提交表单数据之前检查和验证输入数据的过程，以确保数据的正确性、完整性和安全性。
	passwordLoginForm := forms.PassWordLoginForm{}
	if err := c.ShouldBind(&passwordLoginForm); err != nil {
		// 如果绑定或验证失败，处理错误
		HandleValidatorError(c, err)
		return
	}
	if !store.Verify(passwordLoginForm.CaptchaId, passwordLoginForm.Captcha, true) {
		c.JSON(http.StatusBadRequest, gin.H{
			"captcha": "验证码错误",
		})
		return
	}
	// 拨号连接用户 gRPC 服务器
	userConn, err := grpc.Dial(fmt.Sprintf("%s:%d", global.ServerConfig.UserSrvInfo.Host,
		global.ServerConfig.UserSrvInfo.Port), grpc.WithInsecure())
	if err != nil {
		zap.S().Errorw("[GetUserList]连接【用户服务失败】",
			"msg", err.Error()) // 记录连接用户服务失败的错误信息
	}

	// 生成 gRPC 的客户端并调用接口
	userSrvClient := proto.NewUserClient(userConn)
	//登陆的逻辑

	rsp, err := userSrvClient.GetUserByMobile(context.Background(), &proto.MobileRequest{
		Mobile: passwordLoginForm.Mobile,
	})
	if err != nil {
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.NotFound:
				c.JSON(http.StatusBadRequest, map[string]string{
					"mobile": "用户不存在",
				})
			default:
				c.JSON(http.StatusInternalServerError, map[string]string{
					"mobile": "登陆失败",
				})
			}
			return
		}
	} else {
		//只是查询到用户并没有检查密码
		if passRsp, passErr := userSrvClient.CheckPassWord(context.Background(), &proto.PasswordCheckInfo{
			Password:          passwordLoginForm.PassWord,
			EncryptedPassword: rsp.PassWord,
		}); passErr != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{
				"password": "登陆失败",
			})
		} else {
			if passRsp.Success {
				//生成token
				j := middlewares.NewJWT()
				claims := models.CustomClaims{
					ID:          uint(rsp.Id),
					NickName:    rsp.NickName,
					AuthorityId: uint(rsp.Role),
					StandardClaims: jwt.StandardClaims{
						NotBefore: time.Now().Unix(),               //签名的生效时间
						ExpiresAt: time.Now().Unix() + 60*60*24*30, //30天过期
						Issuer:    "llb",
					},
				}
				token, err := j.CreateToken(claims)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"msg": "生成token失败",
					})
					return
				}
				c.JSON(http.StatusOK, map[string]string{
					"msg": "登陆成功",
				})
				c.JSON(http.StatusOK, gin.H{
					"id":         rsp.Id,
					"nick_name":  rsp.NickName,
					"token":      token,
					"expired_at": (time.Now().Unix() + 60*60*24*30) * 1000,
				})
			} else {
				c.JSON(http.StatusOK, map[string]string{
					"msg": "登陆失败",
				})
			}
		}
	}

}

func Register(c *gin.Context) {
	//用户注册
	registerForm := forms.RegisterForm{}
	if err := c.ShouldBind(&registerForm); err != nil {
		HandleValidatorError(c, err)
		return
	}

	//验证码
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", global.ServerConfig.RedisInfo.Host, global.ServerConfig.RedisInfo.Port),
	})
	value, err := rdb.Get(context.Background(), registerForm.Mobile).Result()
	if err == redis.Nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": "验证码错误",
		})
		return
	} else {
		if value != registerForm.Code {
			c.JSON(http.StatusBadRequest, gin.H{
				"code": "验证码错误",
			})
			return
		}
	}

	user, err := global.UserSrvClient.CreateUser(context.Background(), &proto.CreateUserInfo{
		NickName: registerForm.Mobile,
		PassWord: registerForm.PassWord,
		Mobile:   registerForm.Mobile,
	})

	if err != nil {
		zap.S().Errorf("[Register] 查询 【新建用户失败】失败: %s", err.Error())
		HandleGrpcErrorToHttp(err, c)
		return
	}

	j := middlewares.NewJWT()
	claims := models.CustomClaims{
		ID:          uint(user.Id),
		NickName:    user.NickName,
		AuthorityId: uint(user.Role),
		StandardClaims: jwt.StandardClaims{
			NotBefore: time.Now().Unix(),               //签名的生效时间
			ExpiresAt: time.Now().Unix() + 60*60*24*30, //30天过期
			Issuer:    "imooc",
		},
	}
	token, err := j.CreateToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "生成token失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.Id,
		"nick_name":  user.NickName,
		"token":      token,
		"expired_at": (time.Now().Unix() + 60*60*24*30) * 1000,
	})
}

//func GetUserDetail(ctx *gin.Context){
//	claims, _ := ctx.Get("claims")
//	currentUser := claims.(*models.CustomClaims)
//	zap.S().Infof("访问用户: %d", currentUser.ID)
//
//	rsp, err := global.UserSrvClient.GetUserById(context.Background(), &proto.IdRequest{
//		Id: int32(currentUser.ID),
//	})
//	if err != nil {
//		HandleGrpcErrorToHttp(err, ctx)
//		return
//	}
//	ctx.JSON(http.StatusOK, gin.H{
//		"name":rsp.NickName,
//		"birthday": time.Unix(int64(rsp.BirthDay), 0).Format("2006-01-02"),
//		"gender":rsp.Gender,
//		"mobile":rsp.Mobile,
//	})
//}
//
//
//func UpdateUser(ctx *gin.Context){
//	updateUserForm := forms.UpdateUserForm{}
//	if err := ctx.ShouldBind(&updateUserForm); err != nil {
//		HandleValidatorError(ctx, err)
//		return
//	}
//
//	claims, _ := ctx.Get("claims")
//	currentUser := claims.(*models.CustomClaims)
//	zap.S().Infof("访问用户: %d", currentUser.ID)
//
//	//将前端传递过来的日期格式转换成int
//	loc, _ := time.LoadLocation("Local") //local的L必须大写
//	birthDay, _ := time.ParseInLocation("2006-01-02", updateUserForm.Birthday, loc)
//	_, err := global.UserSrvClient.UpdateUser(context.Background(), &proto.UpdateUserInfo{
//		Id:       int32(currentUser.ID),
//		NickName: updateUserForm.Name,
//		Gender:   updateUserForm.Gender,
//		BirthDay: uint64(birthDay.Unix()),
//	})
//	if err != nil {
//		HandleGrpcErrorToHttp(err, ctx)
//		return
//	}
//	ctx.JSON(http.StatusOK, gin.H{})
//}

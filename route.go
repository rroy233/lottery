package main

import "net/http"

func init()  {

	//视图
	http.HandleFunc("/login",viewUserLogin)
	http.HandleFunc("/register",viewUserRegister)
	http.HandleFunc("/",viewUserIndex)
	http.HandleFunc("/admin/",viewAdminDash)
	http.HandleFunc("/admin/lottery",viewAdminLottery)
	http.HandleFunc("/admin/gifts",viewAdminGifts)
	http.HandleFunc("/admin/user",viewAdminUser)
	http.HandleFunc("/admin/setting",viewAdminSetting)



	http.HandleFunc("/sso/callback",SSOController)
	http.HandleFunc("/auth/ticket",adminMakeTicket)
	http.HandleFunc("/auth/login",LoginController)
	http.HandleFunc("/auth/logout",LogoutController)
	http.HandleFunc("/auth/reg",RegController)

	http.HandleFunc("/qrcode", QrCodeController)//二维码生成地址

	http.HandleFunc("/user/act_info",actInfo)
	http.HandleFunc("/user/heartbeat",heartbeat)
	http.HandleFunc("/user/lottery",lottery)
	http.HandleFunc("/user/my",myGifts)


	http.HandleFunc("/admin/index/act_info",adminActInfo)
	http.HandleFunc("/admin/index/check",dashboardCheck)

	http.HandleFunc("/admin/user/add",adminAddUser)
	http.HandleFunc("/admin/user/all",adminGetAllUser)
	http.HandleFunc("/admin/user/get",adminGetUser)
	http.HandleFunc("/admin/user/edit",adminEditUser)
	http.HandleFunc("/admin/user/del",adminDelUser)
	http.HandleFunc("/admin/user/bulk_del",adminBulkDelUser)
	http.HandleFunc("/admin/user/bulk_add",adminBulkAddUser)

	http.HandleFunc("/admin/act/get",adminGetAct)
	http.HandleFunc("/admin/act/edit",adminEditAct)
	http.HandleFunc("/admin/act/open",openAct)
	http.HandleFunc("/admin/act/close",CloseAct)
	http.HandleFunc("/admin/act/reset",ResetAct)
	http.HandleFunc("/admin/act/lucky_list",adminGetLuckyList)
	http.HandleFunc("/admin/act/lucky_list/export",adminExportLuckyList)

	http.HandleFunc("/admin/gifts/all",adminGetAllGifts)
	http.HandleFunc("/admin/gifts/namelist",adminGetGiftsNameList)
	http.HandleFunc("/admin/gifts/add",adminAddGifts)
	http.HandleFunc("/admin/gifts/get",adminGetGifts)
	http.HandleFunc("/admin/gifts/edit",adminEditGifts)
	http.HandleFunc("/admin/gifts/del",adminDelGifts)

}

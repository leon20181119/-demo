package main

import (
	"colly"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/shopspring/decimal"

	uuid "github.com/satori/go.uuid"
	"gitlab.99safe.org/gather-products/data/model"
)

//备注：完成数据导入后，修改config/server.json  RedisServer "DB": "1",
func main() {

	db := getDB()
	defer db.Close()

	//1，爬取数据到数据数据库
	// spider(db)

	//2.去重并整理商品价格
	// var goodsItem []model.GoodsItem
	// db.Find(&goodsItem)

	// newGoodsItems, err := countProductPrice(goodsItem)
	// if err != nil {
	// 	Log.Error(err)
	// 	return
	// }
	// Log.Debug("goodsItem len:", len(newGoodsItems))

	//3.修改商品url为cdn的url
	err = downloadAndUpAllImg(newGoodsItems, db)
	if err != nil {
		Log.Error(err)
		return
	}

}

func getDB() *gorm.DB {
	db, err := gorm.Open("mysql", "自己写数据库用户名:自己写数据库密码@/spider?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		Log.Error(err)
		return db
	}
	return db
}

//爬取数据
func spider(db *gorm.DB) {

	if !db.HasTable(&model.GoodsItem{}) {
		db.Set("gorm:table_options", "ENGINE=InnoDB").CreateTable(&model.GoodsItem{})
	}

	fmt.Println("spider  start...")
	c := colly.NewCollector(
		colly.Async(true), //抓取异步
		colly.UserAgent("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"), //表示限制只抓取域名是douban(域名后缀和二级域名不限制)的地址，当然还支持正则匹配某些符合的 URLS，具体的可以看官方文档。
	)

	c.Limit(&colly.LimitRule{DomainGlob: "*.sundan.*", Parallelism: 5}) //限制并发数为5

	//请求前
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	//请求过程中发生错误
	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong:", err)
	})

	//如果收到的响应内容是HTML调用它
	//抓取条目信息的逻辑
	c.OnHTML(".clearfix", func(e *colly.HTMLElement) {

		var productList []model.GoodsItem

		//循环解析每一个li
		e.ForEach("ul>li", func(i int, sub *colly.HTMLElement) {

			//取到button 里面的data-compare
			buttonData := sub.DOM.Find("button").AttrOr("data-compare", "")

			var productStruct ProductStruct
			if buttonData != "" {
				buttonData = strings.Replace(buttonData, "type_id", "\"type_id\"", -1)
				buttonData = strings.Replace(buttonData, "goods_id", "\"goods_id\"", -1)
				buttonData = strings.Replace(buttonData, "product_id", "\"product_id\"", -1)
				buttonData = strings.Replace(buttonData, "img", "\"img\"", -1)
				buttonData = strings.Replace(buttonData, "name", "\"name\"", -1)
				buttonData = strings.Replace(buttonData, "price", "\"price\"", -1)

				if err := json.Unmarshal([]byte(buttonData), &productStruct); err == nil {
					var goodsItem model.GoodsItem
					goodsItem.ProductName = productStruct.Name //商品名称
					goodsItem.Url = productStruct.Img          //图片url

					//处理price
					if productStruct.Price != "" {
						//去掉￥，并转成decimal类型
						priceTmp, _ := decimal.NewFromString(strings.Trim(strings.Replace(productStruct.Price, "￥", "", -1), ""))
						if err != nil {
							Log.Error(err)
						}
						goodsItem.Amount = priceTmp //价格
					}

					productList = append(productList, goodsItem)

				} else {
					return
				}
			}

		})

		if len(productList) > 0 {

			//处理重复及价格
			productList, err := countProductPrice(productList)
			if err != nil {
				Log.Error(err)
				return
			}

			for i := 0; i < len(productList); i++ {

				// //获取cdn的URL
				// imgName, err := downloadImg(productList[i].Url)
				// if err != nil {
				// 	Log.Error(err)
				// 	return
				// }
				// imgUrl, err := UploadImg(imgName)
				// if err != nil {
				// 	Log.Error(err)
				// 	return
				// }
				// Log.Debug("url from cdb:", imgUrl)
				// productList[i].Url = imgUrl

				fmt.Println("DB insert============", productList[i])
				db.Create(&productList[i])
			}
		}
	})

	//在OnXML/OnHTML回调完成后调用。不过官网写的是Called after OnXML callbacks，实际上对于OnHTML也有效，大家可以注意一下。
	// c.OnScraped(func(r *colly.Response) {
	// 	fmt.Println("Finished", r.Request.URL)
	// })

	//暂时不要删除，一个一个的爬取数据
	c.Visit("https://www.sundan.com/gallery-201.html") //空调
	// c.Visit("https://www.sundan.com/gallery-118.html") //手机
	// c.Visit("https://www.sundan.com/gallery-30.html") //电视
	// c.Visit("https://www.sundan.com/gallery-57.html") //音响
	// c.Visit("https://www.sundan.com/gallery-420.html") //相机
	// c.Visit("https://www.sundan.com/gallery-129.html") //运动配件
	// c.Visit("https://www.sundan.com/gallery-42.html") //口腔护理
	// c.Visit("https://www.sundan.com/gallery-84.html") //移动电源
	// c.Visit("https://www.sundan.com/gallery-85.html") //手机壳
	// c.Visit("https://www.sundan.com/gallery-116.html") //手机贴膜
	// c.Visit("https://www.sundan.com/gallery-227.html") //耳机
	// c.Visit("https://www.sundan.com/gallery-245.html") //mp3/mp4
	// c.Visit("https://www.sundan.com/gallery-407.html") //智能手表手环
	// c.Visit("https://www.sundan.com/gallery-95.html") //游戏电玩
	// c.Visit("https://www.sundan.com/gallery-40.html") //笔记本
	// c.Visit("https://www.sundan.com/gallery-82.html") //平板
	// c.Visit("https://www.sundan.com/gallery-209.html") //咖啡机
	// c.Visit("https://www.sundan.com/gallery-36.html") //剃须刀
	//c.Visit("https://www.sundan.com/gallery-95.html ")  //游戏电玩
	// c.Visit("https://www.sundan.com/gallery-456.html ") //翻译机

	c.Wait()
}

//处理重复数据并重新计算最小最大价格
func countProductPrice(dataFromDB []model.GoodsItem) (dataToDoDB []model.GoodsItem, err error) {

	dataToDoDB = dataFromDB
	if len(dataFromDB) > 0 {
		Log.Debug("curr record num:", len(dataFromDB))
		//去重复
		for _, i := range dataFromDB {
			if len(dataToDoDB) == 0 {
				dataToDoDB = append(dataToDoDB, i)
			} else {
				for k, v := range dataToDoDB {
					if (strings.Trim(i.ProductName, "") == strings.Trim(v.ProductName, "") && strings.Trim(i.Url, "") == strings.Trim(v.Url, "")) ||
						i.Amount.Equal(v.Amount) {
						break
					}

					if k == len(dataToDoDB)-1 {
						dataToDoDB = append(dataToDoDB, i)
					}
				}
			}
		}
		fmt.Println("after repeat num:", len(dataToDoDB))

		//1,处理每一件商品的价格区间，连续且不可重复
		if len(dataToDoDB) > 0 {
			sort.Sort(model.GoodsItemSlice(dataToDoDB)) //降序排序

			//设置每一件商品的最低价与最高价
			for i := 0; i < len(dataToDoDB)-1; i++ {
				dataToDoDB[i].MinAmount = dataToDoDB[i].Amount
				dataToDoDB[i].MaxAmount = dataToDoDB[i].Amount.Add(decimal.New(1000, 0))
			}

			//从大到小修改每一件商品的最大价格为上一个商品的最小价格-1
			for k := 1; k < len(dataToDoDB)-1; k++ {
				dataToDoDB[k].MaxAmount = dataToDoDB[k-1].MinAmount.Sub(decimal.New(1, 0))
			}
			//最后一个商品的最大金额为倒数第二的最小金额-1
			dataToDoDB[len(dataToDoDB)-1].MaxAmount = dataToDoDB[len(dataToDoDB)-2].MinAmount.Sub(decimal.New(1, 0))
		}
	}
	return dataToDoDB, nil
}

//获取图片
func HttpGetImg(url string, connTimeoutMs int, serveTimeoutMs int) ([]byte, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Duration(connTimeoutMs)*time.Millisecond)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(time.Now().Add(time.Duration(serveTimeoutMs) * time.Millisecond))
				return c, nil
			},
		},
	}

	reqest, _ := http.NewRequest(http.MethodGet, url, nil)
	response, err := client.Do(reqest)
	if err != nil {
		err = errors.New(fmt.Sprintf("http failed, GET url:%s, reason:%s", url, err.Error()))
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("http status code error, GET url:%s, code:%d", url, response.StatusCode))
		return nil, err
	}

	res_body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err = errors.New(fmt.Sprintf("cannot read http response, GET url:%s, reason:%s", url, err.Error()))
		return nil, err
	}

	return res_body, nil
}

//下载图片到本地
func downloadImg(imgUrl string) (imgName string, err error) {

	imgInfo, err := HttpGetImg(imgUrl, int(3*time.Second), int(3*time.Second))
	if err != nil {
		Log.Error(err)
		return "", err
	}

	//创建目录(判断是否存在)
	dir := "img"
	checkFileIsExist := checkFileIsExist(dir)
	if !checkFileIsExist {
		os.MkdirAll(dir, os.ModePerm)
	}

	//生成图片名称
	u1 := uuid.NewV4()

	Log.Debug("UUIDv4: %s\n", u1)

	out, err := os.Create(dir + "/" + u1.String() + ".jpg")
	defer out.Close()
	if err != nil {
		Log.Error(err)
	}

	out.Write(imgInfo)

	Log.Debug("finished!!")

	return u1.String() + ".jpg", nil
}

//UploadImg 上传图片到服务器
func UploadImg(imgName string) (imgUrl string, err error) {
	mediaService := NewMediaService()

	path, err := mediaService.Upload("自己写服务器地址/"+imgName, "自己写本地地址/"+imgName, gocloud.ImageType)
	if err != nil {
		Log.Error(err)
		return "", err
	}
	Log.Debug("img cdn path:", path)
	return path, nil

}

//操作单张图片
func downloadUpImg(goodsItems model.GoodsItem, db *gorm.DB) (err error) {
	Log.Debug("download img:", goodsItems)
	//先下载单张图片
	imgName, err := downloadImg(goodsItems.Url)
	if err != nil {
		Log.Error(err)
		return err
	}
	Log.Debug("upload img:", imgName)
	//上传该图片
	imgUrl, err := UploadImg(imgName)
	if err != nil {
		Log.Error(err)
		return err
	}

	Log.Debug("update  img:", imgUrl)
	//执行更新数据库操作
	attr := make(map[string]interface{})
	attr["url"] = imgUrl
	err = db.Model(&model.GoodsItem{}).Where("ID = ?", goodsItems.ID).Updates(attr).Error
	if err != nil {
		Log.Error(err)
		return err
	}

	return nil

}

//操作数据库的多张图片
func downloadAndUpAllImg(goodsItems []model.GoodsItem, db *gorm.DB) (err error) {
	if len(goodsItems) > 0 {
		for i := 0; i < len(goodsItems); i++ {
			//下载并上传
			err = downloadUpImg(goodsItems[i], db)
			if err != nil {
				Log.Error(err)
				return err
			}
		}
	}
	return nil
}

//checkFileIsExist 检查目录是否存在
func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		Log.Debug(filename + " not exist")
		exist = false
	}
	return exist
}

type ProductStruct struct {
	TypeID    string `json:"type_id"`
	GoodsID   string `json:"goods_id"`
	ProductID string `json:"product_id"`
	Img       string `json:"img"`
	Name      string `json:"name"`
	Price     string `json:"price"`
}

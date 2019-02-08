package service

import (
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/puti-projects/puti/internal/common/config"
	"github.com/puti-projects/puti/internal/common/model"
	"github.com/puti-projects/puti/internal/common/utils"
	"github.com/puti-projects/puti/internal/pkg/logger"
	optionCache "github.com/puti-projects/puti/internal/pkg/option"
)

// List post list
type List struct {
	Lock  *sync.Mutex
	IDMap map[uint64]*model.ShowArticle
}

// GetArticleListByTaxonomy get articles list
func GetArticleListByTaxonomy(currentPage int, taxonomyType, taxonomySlug, keyword string) (termName string, articleResult []*model.ShowArticle, pagination *utils.Pagination, err error) {
	// get articles data
	pageSize, _ := strconv.Atoi(optionCache.Options.Get("posts_per_page"))
	offset := (currentPage - 1) * pageSize
	count := 0

	// get term id
	var termTaxonomyID uint64
	getTermTaxonomyID := model.DB.Local.Table("pt_term t").
		Select("t.`name`, tt.`term_taxonomy_id`").
		Joins("INNER JOIN pt_term_taxonomy tt ON tt.term_id = t.term_id").
		Where("t.`slug` = ? AND tt.`taxonomy` = ?", taxonomySlug, taxonomyType).
		Row()
	getTermTaxonomyID.Scan(&termName, &termTaxonomyID)

	// get article list
	where := "p.`deleted_time` IS NULL AND p.`post_type` = ? AND p.`parent_id` = ? AND p.`status` = ? AND tr.`term_taxonomy_id` = ?"
	whereArgs := []interface{}{model.PostTypeArticle, 0, model.PostStatusPublish, termTaxonomyID}
	if "" != keyword {
		where += " AND p.`title` LIKE ?"
		whereArgs = append(whereArgs, "%"+keyword+"%")
	}

	articles := []*model.PostModel{}
	result := model.DB.Local.Table("pt_post p").
		Select("p.`id`, p.`title`, p.`if_top`, p.`content_html`, p.`guid`, p.`cover_picture`, p.`comment_count`, p.`view_count`, p.`posted_time`").
		Joins("INNER JOIN pt_term_relationships tr ON tr.object_id = p.id").
		Unscoped().
		Where(where, whereArgs...).Count(&count).
		Order("p.`if_top` DESC, p.`posted_time` DESC").
		Offset(offset).Limit(pageSize).
		Find(&articles)

	if err := result.Error; err != nil {
		logger.Errorf("get articles failed. %s", err)
	}

	// get pagination
	pagination = utils.GetPagination(count, currentPage, pageSize, 0)

	// handle articles data
	articleResult = make([]*model.ShowArticle, 0)
	ids := []uint64{}
	for _, article := range articles {
		ids = append(ids, article.ID)
	}

	wg := sync.WaitGroup{}
	articleList := List{
		Lock:  new(sync.Mutex),
		IDMap: make(map[uint64]*model.ShowArticle, len(articles)),
	}

	errChan := make(chan error, 1)
	finished := make(chan bool, 1)

	siteURL := optionCache.Options.Get("site_url")
	for _, a := range articles {
		wg.Add(1)
		go func(a *model.PostModel) {
			defer wg.Done()

			var ifTop = false
			if a.IfTop == 1 {
				ifTop = true
			}

			articleCategory, articleTag := getArticleTaxonomyInfo(a.ID, siteURL)

			articleList.Lock.Lock()
			defer articleList.Lock.Unlock()
			articleList.IDMap[a.ID] = &model.ShowArticle{
				ID:           a.ID,
				Title:        template.HTML(a.Title),
				IfTop:        ifTop,
				Abstract:     getArticleAbstract(a.ContentHTML),
				GUID:         a.GUID,
				CoverPicture: a.CoverPicture,
				CommentCount: a.CommentCount,
				ViewCount:    a.ViewCount,
				PostedTime:   a.PostDate.In(config.TimeLoc()).Format("2006-01-02 15:04"),
				Tags:         articleTag,
				Categories:   articleCategory,
			}
		}(a)
	}

	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case err := <-errChan:
		return "", nil, nil, err
	}

	for _, id := range ids {
		articleResult = append(articleResult, articleList.IDMap[id])
	}

	return
}

// GetArticleList get articles list
func GetArticleList(currentPage int, keyword string) (articleResult []*model.ShowArticle, pagination *utils.Pagination, err error) {
	// get articles data
	pageSize, _ := strconv.Atoi(optionCache.Options.Get("posts_per_page"))
	offset := (currentPage - 1) * pageSize
	count := 0

	where := "`post_type` = ? AND `parent_id` = ? AND `status` = ?"
	whereArgs := []interface{}{model.PostTypeArticle, 0, model.PostStatusPublish}
	if "" != keyword {
		where += " AND `title` LIKE ?"
		whereArgs = append(whereArgs, "%"+keyword+"%")
	}

	articles := []*model.PostModel{}
	result := model.DB.Local.Model(&model.PostModel{}).
		Select("`id`, `title`, `if_top`, `content_html`, `guid`, `cover_picture`, `comment_count`, `view_count`, `posted_time`").
		Where(where, whereArgs...).Count(&count).
		Order("`if_top` DESC, `posted_time` DESC").
		Offset(offset).Limit(pageSize).
		Find(&articles)

	if err := result.Error; err != nil {
		logger.Errorf("get articles failed. %s", err)
	}

	// get pagination
	pagination = utils.GetPagination(count, currentPage, pageSize, 0)

	// handle articles data
	articleResult = make([]*model.ShowArticle, 0)
	ids := []uint64{}
	for _, article := range articles {
		ids = append(ids, article.ID)
	}

	wg := sync.WaitGroup{}
	articleList := List{
		Lock:  new(sync.Mutex),
		IDMap: make(map[uint64]*model.ShowArticle, len(articles)),
	}

	errChan := make(chan error, 1)
	finished := make(chan bool, 1)

	siteURL := optionCache.Options.Get("site_url")
	for _, a := range articles {
		wg.Add(1)
		go func(a *model.PostModel) {
			defer wg.Done()

			var ifTop = false
			if a.IfTop == 1 {
				ifTop = true
			}

			articleCategory, articleTag := getArticleTaxonomyInfo(a.ID, siteURL)

			articleList.Lock.Lock()
			defer articleList.Lock.Unlock()
			articleList.IDMap[a.ID] = &model.ShowArticle{
				ID:           a.ID,
				Title:        template.HTML(a.Title),
				IfTop:        ifTop,
				Abstract:     getArticleAbstract(a.ContentHTML),
				GUID:         a.GUID,
				CoverPicture: a.CoverPicture,
				CommentCount: a.CommentCount,
				ViewCount:    a.ViewCount,
				PostedTime:   a.PostDate.In(config.TimeLoc()).Format("2006-01-02 15:04"),
				Tags:         articleTag,
				Categories:   articleCategory,
			}
		}(a)
	}

	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case err := <-errChan:
		return nil, nil, err
	}

	for _, id := range ids {
		articleResult = append(articleResult, articleList.IDMap[id])
	}

	return
}

func getArticleTaxonomyInfo(articleID uint64, siteURL string) ([]*model.ShowCategory, []*model.ShowTag) {
	sql := "SELECT t.name, t.slug, tt.taxonomy FROM pt_term t LEFT JOIN pt_term_taxonomy tt ON tt.term_id = t.term_id LEFT JOIN pt_term_relationships tr ON tr.term_taxonomy_id = tt.term_taxonomy_id WHERE tr.object_id = ?"
	rows, _ := model.DB.Local.Raw(sql, articleID).Rows()
	defer rows.Close()

	articleCategory := make([]*model.ShowCategory, 0)
	articleTag := make([]*model.ShowTag, 0)
	var name string
	var slug string
	var taxonomy string
	for rows.Next() {
		rows.Scan(&name, &slug, &taxonomy)
		if taxonomy == "category" {
			articleCategory = append(articleCategory, &model.ShowCategory{Title: name, URL: siteURL + config.PathCategory + "/" + slug})
		}

		if taxonomy == "tag" {
			articleTag = append(articleTag, &model.ShowTag{Title: name, URL: siteURL + config.PathTag + "/" + slug})
		}
	}

	return articleCategory, articleTag
}

func getArticleAbstract(content string) string {
	re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
	content = re.ReplaceAllString(content, "\n")

	re, _ = regexp.Compile("\\s{2,}")
	content = re.ReplaceAllString(content, "\n")

	content = strings.TrimSpace(content)

	abstractRune := []rune(content)
	contentLen := len(abstractRune)
	if contentLen <= 200 {
		return content
	}

	abstract := fmt.Sprintf("%s%s", string(abstractRune[:200]), "...")
	return abstract
}

// GetLatestArticlesList get latest article list for widget
func GetLatestArticlesList(getNums int) ([]*model.ShowWidgetLatestArticles, error) {
	where := "`post_type` = ? AND `parent_id` = ? AND `status` = ?"
	whereArgs := []interface{}{model.PostTypeArticle, 0, model.PostStatusPublish}

	articles := []*model.ShowWidgetLatestArticles{}
	postModel := &model.PostModel{}
	result := model.DB.Local.Table(postModel.TableName()).
		Select("`id`, `title`, `guid`, `comment_count`, `view_count`, `posted_time`").
		Where(where, whereArgs...).
		Order("`posted_time` DESC").
		Limit(getNums).
		Find(&articles)

	if err := result.Error; err != nil {
		return nil, err
	}

	return articles, nil
}

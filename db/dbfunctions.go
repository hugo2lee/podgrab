package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetPodcastByURL(url string, podcast *Podcast) error {
	result := DB.Preload(clause.Associations).Where(&Podcast{URL: url}).First(&podcast)
	return result.Error
}

func GetPodcastsByURLList(urls []string, podcasts *[]Podcast) error {
	result := DB.Preload(clause.Associations).Where("url in ?", urls).First(&podcasts)
	return result.Error
}
func GetAllPodcasts(podcasts *[]Podcast, sorting string) error {
	if sorting == "" {
		sorting = "created_at"
	}
	result := DB.Debug().Order(sorting).Find(&podcasts)
	return result.Error
}
func GetAllPodcastItems(podcasts *[]PodcastItem) error {

	result := DB.Preload("Podcast").Order("pub_date desc").Find(&podcasts)
	return result.Error
}
func GetPaginatedPodcastItems(page int, count int, downloadedOnly *bool, playedOnly *bool, fromDate time.Time, podcasts *[]PodcastItem, total *int64) error {
	query := DB.Debug().Preload("Podcast")
	if downloadedOnly != nil {
		if *downloadedOnly {
			query = query.Where("download_status=?", Downloaded)
		} else {
			query = query.Where("download_status!=?", Downloaded)
		}
	}
	if playedOnly != nil {
		if *playedOnly {
			query = query.Where("is_played=?", 1)
		} else {
			query = query.Where("is_played=?", 0)
		}
	}
	if (fromDate != time.Time{}) {
		query = query.Where("pub_date>=?", fromDate)
	}

	result := query.Limit(count).Offset((page - 1) * count).Order("pub_date desc").Find(&podcasts)

	query.Count(total)

	return result.Error
}
func GetPodcastById(id string, podcast *Podcast) error {

	result := DB.Preload("PodcastItems", func(db *gorm.DB) *gorm.DB {
		return db.Order("podcast_items.pub_date DESC")
	}).First(&podcast, "id=?", id)
	return result.Error
}

func GetPodcastItemById(id string, podcastItem *PodcastItem) error {

	result := DB.Preload(clause.Associations).First(&podcastItem, "id=?", id)
	return result.Error
}
func DeletePodcastItemById(id string) error {

	result := DB.Where("id=?", id).Delete(&PodcastItem{})
	return result.Error
}
func DeletePodcastById(id string) error {

	result := DB.Where("id=?", id).Delete(&Podcast{})
	return result.Error
}

func GetAllPodcastItemsByPodcastId(podcastId string, podcastItems *[]PodcastItem) error {

	result := DB.Preload(clause.Associations).Where(&PodcastItem{PodcastID: podcastId}).Find(&podcastItems)
	return result.Error
}

func SetAllEpisodesToDownload(podcastId string) error {
	result := DB.Debug().Model(PodcastItem{}).Where(&PodcastItem{PodcastID: podcastId, DownloadStatus: Deleted}).Update("download_status", NotDownloaded)
	return result.Error
}
func UpdateLastEpisodeDateForPodcast(podcastId string, lastEpisode time.Time) error {
	result := DB.Debug().Model(Podcast{}).Where("id=?", podcastId).Update("last_episode", lastEpisode)
	return result.Error
}

func GetAllPodcastItemsToBeDownloaded() (*[]PodcastItem, error) {
	var podcastItems []PodcastItem
	result := DB.Debug().Preload(clause.Associations).Where("download_status=?", NotDownloaded).Find(&podcastItems)
	//fmt.Println("To be downloaded : " + string(len(podcastItems)))
	return &podcastItems, result.Error
}
func GetAllPodcastItemsAlreadyDownloaded() (*[]PodcastItem, error) {
	var podcastItems []PodcastItem
	result := DB.Debug().Preload(clause.Associations).Where("download_status=?", Downloaded).Find(&podcastItems)
	return &podcastItems, result.Error
}

func GetPodcastEpisodeStats() (*[]PodcastItemStatsModel, error) {
	var stats []PodcastItemStatsModel
	result := DB.Model(&PodcastItem{}).Select("download_status,podcast_id, count(1) as count").Group("podcast_id,download_status").Find(&stats)
	return &stats, result.Error
}

func GetEpisodeNumber(podcastItemId, podcastId string) (int, error) {
	var id string
	var sequence int
	row := DB.Raw(`;With cte as(
		SELECT 
			id, 
			RANK() OVER (ORDER BY pub_date) as sequence 
		FROM 
			podcast_items
		WHERE
			podcast_id=?
	)
	select * 
	from cte 
	where id = ?
	`, podcastId, podcastItemId).Row()
	error := row.Scan(&id, &sequence)
	return sequence, error
}

func ForceSetLastEpisodeDate(podcastId string) {
	DB.Exec("update podcasts set last_episode = (select max(pi.pub_date) from podcast_items pi where pi.podcast_id = @id) where id = @id", sql.Named("id", podcastId))
}

func GetPodcastItemsByPodcastIdAndGUIDs(podcastId string, guids []string) (*[]PodcastItem, error) {
	var podcastItems []PodcastItem
	result := DB.Preload(clause.Associations).Where(&PodcastItem{PodcastID: podcastId}).Where("guid IN ?", guids).Find(&podcastItems)
	return &podcastItems, result.Error
}
func GetPodcastItemByPodcastIdAndGUID(podcastId string, guid string, podcastItem *PodcastItem) error {

	result := DB.Preload(clause.Associations).Where(&PodcastItem{PodcastID: podcastId, GUID: guid}).First(&podcastItem)
	return result.Error
}
func GetPodcastByTitleAndAuthor(title string, author string, podcast *Podcast) error {

	result := DB.Preload(clause.Associations).Where(&Podcast{Title: title, Author: author}).First(&podcast)
	return result.Error
}

func CreatePodcast(podcast *Podcast) error {
	tx := DB.Create(&podcast)
	return tx.Error
}

func CreatePodcastItem(podcastItem *PodcastItem) error {
	tx := DB.Omit("Podcast").Create(&podcastItem)
	return tx.Error
}
func UpdatePodcastItem(podcastItem *PodcastItem) error {
	tx := DB.Omit("Podcast").Save(&podcastItem)
	return tx.Error
}
func UpdateSettings(setting *Setting) error {
	tx := DB.Save(&setting)
	return tx.Error
}
func GetOrCreateSetting() *Setting {
	var setting Setting
	result := DB.First(&setting)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		DB.Save(&Setting{})
		DB.First(&setting)
	}
	return &setting
}

func GetLock(name string) *JobLock {
	var jobLock JobLock
	result := DB.Where("name = ?", name).First(&jobLock)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return &JobLock{
			Name: name,
		}
	}
	return &jobLock
}
func Lock(name string, duration int) {
	jobLock := GetLock(name)
	if jobLock == nil {
		jobLock = &JobLock{
			Name: name,
		}
	}
	jobLock.Duration = duration
	jobLock.Date = time.Now()
	if jobLock.ID == "" {
		DB.Create(&jobLock)
	} else {
		DB.Save(&jobLock)
	}
}
func Unlock(name string) {
	jobLock := GetLock(name)
	if jobLock == nil {
		return
	}
	jobLock.Duration = 0
	jobLock.Date = time.Time{}
	DB.Save(&jobLock)
}

func UnlockMissedJobs() {
	var jobLocks []JobLock

	result := DB.Debug().Find(&jobLocks)
	if result.Error != nil {
		return
	}
	for _, job := range jobLocks {
		if (job.Date == time.Time{}) {
			continue
		}
		var duration time.Duration
		duration = time.Duration(job.Duration)
		d := job.Date.Add(time.Minute * duration)
		if d.Before(time.Now()) {
			fmt.Println(job.Name + " is unlocked")
			Unlock(job.Name)
		}
	}
}

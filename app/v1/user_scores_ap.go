package v1

import (
	"fmt"
	"strings"

	"gopkg.in/thehowl/go-osuapi.v1"
	"github.com/osuYozora/api/common"
	"zxq.co/x/getrank"
)

type userScoreAuto struct {
	Score
	Beatmap beatmap `json:"beatmap"`
}

type userScoresResponseAuto struct {
	common.ResponseBase
	Scores []userScoreAuto `json:"scores"`
}

const userScoreSelectBaseAp = `
SELECT
	scores_auto.id, scores_auto.beatmap_md5, scores_auto.score,
	scores_auto.max_combo, scores_auto.full_combo, scores_auto.mods,
	scores_auto.300_count, scores_auto.100_count, scores_auto.50_count,
	scores_auto.gekis_count, scores_auto.katus_count, scores_auto.misses_count,
	scores_auto.time, scores_auto.play_mode, scores_auto.accuracy, scores_auto.pp,
	scores_auto.completed,

	beatmaps.beatmap_id, beatmaps.beatmapset_id, beatmaps.beatmap_md5,
	beatmaps.song_name, beatmaps.ar, beatmaps.od, beatmaps.difficulty_std,
	beatmaps.difficulty_taiko, beatmaps.difficulty_ctb, beatmaps.difficulty_mania,
	beatmaps.max_combo, beatmaps.hit_length, beatmaps.ranked,
	beatmaps.ranked_status_freezed, beatmaps.latest_update
FROM scores_auto
INNER JOIN beatmaps ON beatmaps.beatmap_md5 = scores_auto.beatmap_md5
INNER JOIN users ON users.id = scores_auto.userid
`

// UserScoresBestGET retrieves the best scores of an user, sorted by PP if
// mode is standard and sorted by ranked score otherwise.
func UserScoresBestAPGET(md common.MethodData) common.CodeMessager {
	cm, wc, param := whereClauseUser(md, "users")
	if cm != nil {
		return *cm
	}
	mc := genModeClauseAp(md)
	// For all modes that have PP, we leave out 0 PP scores_auto.
	if getMode(md.Query("mode")) != "ctb" {
		mc += " AND scores_auto.pp > 0"
	}
	return scoresPutsAp(md, fmt.Sprintf(
		`WHERE
			scores_auto.completed = '3'
			AND %s
			%s
			AND `+md.User.OnlyUserPublic(true)+`
		ORDER BY scores_auto.pp DESC, scores_auto.score DESC %s`,
		wc, mc, common.Paginate(md.Query("p"), md.Query("l"), 100),
	), param)
}

// UserScoresRecentGET retrieves an user's latest scores_auto.
func UserScoresRecentAPGET(md common.MethodData) common.CodeMessager {
	cm, wc, param := whereClauseUser(md, "users")
	if cm != nil {
		return *cm
	}
	return scoresPutsAp(md, fmt.Sprintf(
		`WHERE
			%s
			%s
			AND `+md.User.OnlyUserPublic(true)+`
		ORDER BY scores_auto.id DESC %s`,
		wc, genModeClauseAp(md), common.Paginate(md.Query("p"), md.Query("l"), 100),
	), param)
}

func scoresPutsAp(md common.MethodData, whereClause string, params ...interface{}) common.CodeMessager {
	rows, err := md.DB.Query(userScoreSelectBaseAp+whereClause, params...)
	if err != nil {
		md.Err(err)
		return Err500
	}
	var scores []userScoreAuto
	for rows.Next() {
		var (
			us userScoreAuto
			b  beatmap
		)
		err = rows.Scan(
			&us.ID, &us.BeatmapMD5, &us.Score.Score,
			&us.MaxCombo, &us.FullCombo, &us.Mods,
			&us.Count300, &us.Count100, &us.Count50,
			&us.CountGeki, &us.CountKatu, &us.CountMiss,
			&us.Time, &us.PlayMode, &us.Accuracy, &us.PP,
			&us.Completed,

			&b.BeatmapID, &b.BeatmapsetID, &b.BeatmapMD5,
			&b.SongName, &b.AR, &b.OD, &b.Diff2.STD,
			&b.Diff2.Taiko, &b.Diff2.CTB, &b.Diff2.Mania,
			&b.MaxCombo, &b.HitLength, &b.Ranked,
			&b.RankedStatusFrozen, &b.LatestUpdate,
		)
		if err != nil {
			md.Err(err)
			return Err500
		}
		b.Difficulty = b.Diff2.STD
		us.Beatmap = b
		us.Rank = strings.ToUpper(getrank.GetRank(
			osuapi.Mode(us.PlayMode),
			osuapi.Mods(us.Mods),
			us.Accuracy,
			us.Count300,
			us.Count100,
			us.Count50,
			us.CountMiss,
		))
		scores = append(scores, us)
	}
	r := userScoresResponseAuto{}
	r.Code = 200
	r.Scores = scores
	return r
}

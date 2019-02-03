package v1

import (
	"fmt"
	"strings"

	"gopkg.in/thehowl/go-osuapi.v1"
	"zxq.co/ripple/rippleapi/common"
	"zxq.co/x/getrank"
)

type userScoreRx struct {
	Score
	Beatmap beatmap `json:"beatmap"`
}

type userScoresResponseRx struct {
	common.ResponseBase
	Scores []userScoreRx `json:"scores"`
}

const userScoreSelectBaseRelax = `
SELECT
	scores_relax.id, scores_relax.beatmap_md5, scores_relax.score,
	scores_relax.max_combo, scores_relax.full_combo, scores_relax.mods,
	scores_relax.300_count, scores_relax.100_count, scores_relax.50_count,
	scores_relax.gekis_count, scores_relax.katus_count, scores_relax.misses_count,
	scores_relax.time, scores_relax.play_mode, scores_relax.accuracy, scores_relax.pp,
	scores_relax.completed,

	beatmaps.beatmap_id, beatmaps.beatmapset_id, beatmaps.beatmap_md5,
	beatmaps.song_name, beatmaps.ar, beatmaps.od, beatmaps.difficulty_std,
	beatmaps.difficulty_taiko, beatmaps.difficulty_ctb, beatmaps.difficulty_mania,
	beatmaps.max_combo, beatmaps.hit_length, beatmaps.ranked,
	beatmaps.ranked_status_freezed, beatmaps.latest_update
FROM scores
INNER JOIN beatmaps ON beatmaps.beatmap_md5 = scores_relax.beatmap_md5
INNER JOIN users ON users.id = scores_relax.userid
`

// UserScoresBestGET retrieves the best scores of an user, sorted by PP if
// mode is standard and sorted by ranked score otherwise.
func UserScoresBestRelaxGET(md common.MethodData) common.CodeMessager {
	cm, wc, param := whereClauseUser(md, "users")
	if cm != nil {
		return *cm
	}
	mc := genModeClause(md)
	// For all modes that have PP, we leave out 0 PP scores_relax.
	if getMode(md.Query("mode")) != "ctb" {
		mc += " AND scores_relax.pp > 0"
	}
	return scoresPutsRx(md, fmt.Sprintf(
		`WHERE
			scores_relax.completed = '3'
			AND %s
			%s
			AND `+md.User.OnlyUserPublic(true)+`
		ORDER BY scores_relax.pp DESC, scores_relax.score DESC %s`,
		wc, mc, common.Paginate(md.Query("p"), md.Query("l"), 100),
	), param)
}

// UserScoresRecentGET retrieves an user's latest scores_relax.
func UserScoresRecentRelaxGET(md common.MethodData) common.CodeMessager {
	cm, wc, param := whereClauseUser(md, "users")
	if cm != nil {
		return *cm
	}
	return scoresPutsRx(md, fmt.Sprintf(
		`WHERE
			%s
			%s
			AND `+md.User.OnlyUserPublic(true)+`
		ORDER BY scores_relax.id DESC %s`,
		wc, genModeClause(md), common.Paginate(md.Query("p"), md.Query("l"), 100),
	), param)
}

func scoresPutsRx(md common.MethodData, whereClause string, params ...interface{}) common.CodeMessager {
	rows, err := md.DB.Query(userScoreSelectBaseRelax+whereClause, params...)
	if err != nil {
		md.Err(err)
		return Err500
	}
	var scores []userScoreRx
	for rows.Next() {
		var (
			us userScoreRx
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
	r := userScoresResponseRx{}
	r.Code = 200
	r.Scores = scores
	return r
}

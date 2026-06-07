package usecase

import (
	"context"

	articlerepo "github.com/qli8racn/rss-feeder/internal/adapter/driver/readerdb/article"
	"github.com/qli8racn/rss-feeder/internal/domain"
)

type ListMode int

const (
	ListModeUnread    ListMode = iota
	ListModeAll
	ListModeBookmarked
)

type ListUsecase struct {
	articleRepo articlerepo.Repository
}

func NewListUsecase(articleRepo articlerepo.Repository) *ListUsecase {
	return &ListUsecase{articleRepo: articleRepo}
}

func (uc *ListUsecase) Execute(ctx context.Context, mode ListMode) ([]domain.Article, error) {
	var (
		articles []domain.Article
		err      error
	)

	switch mode {
	case ListModeAll:
		articles, err = uc.articleRepo.FindAll(ctx)
	case ListModeBookmarked:
		articles, err = uc.articleRepo.FindBookmarked(ctx)
	default:
		articles, err = uc.articleRepo.FindUnread(ctx)
	}
	if err != nil {
		return nil, err
	}

	for _, a := range articles {
		if a.Read {
			continue
		}
		a.Read = true
		if err := uc.articleRepo.Update(ctx, a); err != nil {
			return nil, err
		}
	}

	return articles, nil
}

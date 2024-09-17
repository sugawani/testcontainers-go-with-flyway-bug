package mutate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"main/util"
)

func Test_Mutate(t *testing.T) {
	cases := map[string]struct {
		want string
	}{
		"create user":  {want: "created user1"},
		"create user2": {want: "created user2"},
		"create user3": {want: "created user3"},
		"create user4": {want: "created user4"},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			db, cleanup := util.NewTestDB(ctx)
			t.Cleanup(cleanup)

			m := NewMutate(db)
			actual, err := m.Execute(tt.want)
			assert.Equal(t, tt.want, actual.Name)
			assert.NoError(t, err)
		})
	}
}

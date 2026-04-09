package media_worker

import (
	"context"
	"testing"

	"github.com/ferdn4ndo/userver-filemgr/lib"
	"github.com/stretchr/testify/require"
)

func TestPool_StopWithoutStart(t *testing.T) {
	t.Parallel()
	p := &Pool{}
	require.NoError(t, p.Stop(context.Background()))
}

func TestPool_DisabledStartStop(t *testing.T) {
	env := lib.Env{MediaProcessingEnabled: false, MediaWorkerCount: 2}
	p := NewPool(env, nil, nil, lib.GetLogger())
	require.NoError(t, p.Start(context.Background()))
	require.NoError(t, p.Stop(context.Background()))
}

func TestPool_ZeroWorkersStartStop(t *testing.T) {
	env := lib.Env{MediaProcessingEnabled: true, MediaWorkerCount: 0}
	p := NewPool(env, nil, nil, lib.GetLogger())
	require.NoError(t, p.Start(context.Background()))
	require.NoError(t, p.Stop(context.Background()))
}

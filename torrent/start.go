package torrent

import (
	"net"
	"time"

	"github.com/cenkalti/rain/torrent/internal/acceptor"
	"github.com/cenkalti/rain/torrent/internal/allocator"
	"github.com/cenkalti/rain/torrent/internal/announcer"
	"github.com/cenkalti/rain/torrent/internal/verifier"
)

func (t *Torrent) start() {
	if t.running() {
		return
	}

	t.log.Info("starting torrent")
	t.errC = make(chan error, 1)

	if t.info != nil {
		if t.data != nil {
			if t.bitfield != nil {
				t.startAcceptor()
				t.startAnnouncers()
				t.startPieceDownloaders()
				t.startUnchokeTimers()
			} else {
				t.startVerifier()
			}
		} else {
			t.startAllocator()
		}
	} else {
		t.startAcceptor()
		t.startAnnouncers()
		t.startInfoDownloaders()
	}
}

func (t *Torrent) startVerifier() {
	t.verifier = verifier.New(t.data.Pieces, t.verifierProgressC, t.verifierResultC)
	go t.verifier.Run()
}

func (t *Torrent) startAllocator() {
	t.allocator = allocator.New(t.info, t.storage, t.allocatorProgressC, t.allocatorResultC)
	go t.allocator.Run()
}

func (t *Torrent) startAnnouncers() {
	if len(t.announcers) > 0 {
		return
	}
	for _, tr := range t.trackersInstances {
		an := announcer.New(tr, t.announcerRequestC, t.completeC, t.addrsFromTrackers, t.log)
		t.announcers = append(t.announcers, an)
		go an.Run()
	}
}

func (t *Torrent) startAcceptor() {
	if t.acceptor != nil {
		return
	}
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: t.port})
	if err != nil {
		t.log.Warningf("cannot listen port %d: %s", t.port, err)
	} else {
		t.log.Notice("Listening peers on tcp://" + listener.Addr().String())
		t.port = listener.Addr().(*net.TCPAddr).Port
		t.acceptor = acceptor.New(listener, t.incomingConnC, t.log)
		go t.acceptor.Run()
	}
}

func (t *Torrent) startUnchokeTimers() {
	if t.unchokeTimer == nil {
		t.unchokeTimer = time.NewTicker(10 * time.Second)
		t.unchokeTimerC = t.unchokeTimer.C
	}
	if t.optimisticUnchokeTimer == nil {
		t.optimisticUnchokeTimer = time.NewTicker(30 * time.Second)
		t.optimisticUnchokeTimerC = t.optimisticUnchokeTimer.C
	}
}

func (t *Torrent) startInfoDownloaders() {
	if t.info != nil {
		return
	}
	running := len(t.infoDownloaders) - len(t.infoDownloadersSnubbed)
	for running < parallelInfoDownloads {
		id := t.nextInfoDownload()
		if id == nil {
			break
		}
		t.log.Debugln("downloading info from", id.Peer.String())
		t.infoDownloaders[id.Peer] = id
		go id.Run()
	}
}

func (t *Torrent) startPieceDownloaders() {
	if t.bitfield == nil {
		return
	}
	running := len(t.pieceDownloaders) - len(t.pieceDownloadersChoked) - len(t.pieceDownloadersSnubbed)
	for running < parallelPieceDownloads {
		// TODO check status of existing downloads
		pd := t.nextPieceDownload()
		if pd == nil {
			break
		}
		t.log.Debugln("downloading piece", pd.Piece.Index, "from", pd.Peer.String())
		t.pieceDownloaders[pd.Peer] = pd
		go pd.Run()
	}
}

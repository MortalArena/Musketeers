# تعليمات الإصلاح اليدوي لـ container.go

## المشكلة
ملف `pkg/session/container.go` يفتقد استيرادات:
- `github.com/MortalArena/Musketeers/pkg/agent/thinking`
- `go.uber.org/zap`

## الحل

افتح الملف `pkg/session/container.go` واذهب للسطور 3-17.

استبدل قسم الاستيرادات الحالي:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)
```

بهذا:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/agent/thinking"
	"github.com/MortalArena/Musketeers/pkg/agent/tools"
	"github.com/MortalArena/Musketeers/pkg/eventbus"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)
```

## التغييرات
- أضف السطر: `"github.com/MortalArena/Musketeers/pkg/agent/thinking"` بعد السطر 12
- أضف السطر: `"go.uber.org/zap"` في نهاية قسم الاستيرادات

## بعد الإصلاح
شغل الأمر:
```bash
go build -v ./...
```

إذا نجح البناء، المشكلة محلولة.

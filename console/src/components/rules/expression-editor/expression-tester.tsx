import { Play } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { expressionApi } from '@/api/expression';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

interface ExpressionTesterProps {
  expression: string;
}

export function ExpressionTester({ expression }: ExpressionTesterProps) {
  const { t } = useTranslation('rules');
  const [context, setContext] = useState('{\n  "user": { "id": "u1", "email": "test@test.com" },\n  "tenant": { "id": "cl" },\n  "campus": { "id": "c1" },\n  "program": { "id": "p1" }\n}');
  const [result, setResult] = useState<{ matched: boolean; result: unknown } | null>(null);
  const [loading, setLoading] = useState(false);

  const handleTest = async () => {
    if (!expression.trim()) return;
    setLoading(true);
    try {
      const parsed = JSON.parse(context) as Record<string, unknown>;
      const res = await expressionApi.test(expression, parsed);
      setResult({ matched: res.matched, result: res.result });
      if (res.error) toast.error(res.error);
    } catch {
      toast.error('Invalid JSON context');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-3 rounded-md border p-4">
      <p className="text-sm font-medium">{t('expression.testTitle')}</p>
      <div className="space-y-2">
        <label className="text-muted-foreground text-xs">{t('expression.testContext')}</label>
        <textarea
          value={context}
          onChange={(e) => setContext(e.target.value)}
          rows={5}
          className="flex w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </div>
      <div className="flex items-center gap-3">
        <Button size="sm" onClick={() => void handleTest()} disabled={loading || !expression}>
          <Play className="mr-1 h-3 w-3" />
          {t('expression.testRun')}
        </Button>
        {result !== null ? (
          <Badge variant={result.matched ? 'success' : 'secondary'}>
            {result.matched ? t('expression.testMatched') : t('expression.testNotMatched')}
          </Badge>
        ) : null}
      </div>
    </div>
  );
}

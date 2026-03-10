import {
  buildResponseExpressionCatalog,
  getResponseExpressionCompletions,
  shouldAutoTriggerCompletion,
} from './response-expression-language';

describe('response expression language', () => {
  it('returns only root symbols on explicit completion from empty input', () => {
    const catalog = buildResponseExpressionCatalog({});

    const result = getResponseExpressionCompletions({
      doc: '',
      cursor: 0,
      catalog,
      explicit: true,
    });

    expect(result?.options.map((option) => option.label)).toEqual(['response']);
  });

  it('returns child field suggestions for object members', () => {
    const catalog = buildResponseExpressionCatalog({
      sampleResponseText: JSON.stringify({
        results: {
          campus_premium_pack: true,
          name: 'North',
        },
      }),
    });

    const result = getResponseExpressionCompletions({
      doc: 'response.body.results.',
      cursor: 'response.body.results.'.length,
      catalog,
      explicit: false,
    });

    expect(result?.options.map((option) => option.label)).toEqual(
      expect.arrayContaining(['campus_premium_pack', 'name']),
    );
  });

  it('returns boolean semantic snippets when using dot on a boolean field', () => {
    const path = 'response.body.results.campus_premium_pack.';
    const catalog = buildResponseExpressionCatalog({
      sampleResponseText: JSON.stringify({
        results: {
          campus_premium_pack: true,
        },
      }),
    });

    const result = getResponseExpressionCompletions({
      doc: path,
      cursor: path.length,
      catalog,
      explicit: false,
    });

    expect(result?.options.map((option) => option.label)).toEqual(
      expect.arrayContaining(['== true', '== false', '!= true', '!= false', '== null', '!= null']),
    );
    expect(result?.options.find((option) => option.label === '== true')?.insertText).toBe(
      'response.body.results.campus_premium_pack == true',
    );
  });

  it('returns rhs literals for boolean equality checks', () => {
    const doc = 'response.body.results.campus_premium_pack == ';
    const catalog = buildResponseExpressionCatalog({
      sampleResponseText: JSON.stringify({
        results: {
          campus_premium_pack: true,
        },
      }),
    });

    const result = getResponseExpressionCompletions({
      doc,
      cursor: doc.length,
      catalog,
      explicit: false,
    });

    expect(result?.options.map((option) => option.label)).toEqual([
      'true',
      'false',
      'null',
      'nil',
    ]);
  });

  it('returns only string-related semantic actions for string fields', () => {
    const path = 'response.body.results.name.';
    const catalog = buildResponseExpressionCatalog({
      sampleResponseText: JSON.stringify({
        results: {
          name: 'North',
        },
      }),
    });

    const result = getResponseExpressionCompletions({
      doc: path,
      cursor: path.length,
      catalog,
      explicit: false,
    });
    const labels = result?.options.map((option) => option.label) ?? [];

    expect(labels).toEqual(
      expect.arrayContaining(['== ""', '!= ""', 'contains ""', 'startsWith ""', 'endsWith ""', 'matches ""']),
    );
    expect(labels).not.toContain('> 0');
  });

  it('suggests Expr-native array operations and never suggests .length', () => {
    const path = 'response.body.results.';
    const catalog = buildResponseExpressionCatalog({
      sampleResponseText: JSON.stringify({
        results: [{ id: 1 }],
      }),
    });

    const result = getResponseExpressionCompletions({
      doc: path,
      cursor: path.length,
      catalog,
      explicit: false,
    });
    const labels = result?.options.map((option) => option.label) ?? [];

    expect(labels).toEqual(expect.arrayContaining(['len(...) > 0', '[0]', '== null', '!= null']));
    expect(labels).not.toContain('.length');
  });

  it('suggests generic and observed response header keys', () => {
    const path = 'response.header.';
    const catalog = buildResponseExpressionCatalog({
      responseHeaderKeys: ['Content-Type', 'X-Decision'],
    });

    const result = getResponseExpressionCompletions({
      doc: path,
      cursor: path.length,
      catalog,
      explicit: false,
    });

    expect(result?.options.map((option) => option.label)).toEqual(
      expect.arrayContaining(['["header-name"]', '["Content-Type"]', '["X-Decision"]']),
    );
  });

  it('returns only logical continuations after a complete clause', () => {
    const doc = 'response.body.results.campus_premium_pack == true ';
    const catalog = buildResponseExpressionCatalog({
      sampleResponseText: JSON.stringify({
        results: {
          campus_premium_pack: true,
        },
      }),
    });

    const result = getResponseExpressionCompletions({
      doc,
      cursor: doc.length,
      catalog,
      explicit: true,
    });

    expect(result?.options.map((option) => option.label)).toEqual(['and', 'or']);
  });

  it('returns only valid operands after and/or', () => {
    const doc = 'response.body.results.campus_premium_pack == true and ';
    const catalog = buildResponseExpressionCatalog({
      sampleResponseText: JSON.stringify({
        results: {
          campus_premium_pack: true,
        },
      }),
    });

    const result = getResponseExpressionCompletions({
      doc,
      cursor: doc.length,
      catalog,
      explicit: false,
    });
    const labels = result?.options.map((option) => option.label) ?? [];

    expect(labels).toEqual(expect.arrayContaining(['response', 'not']));
    expect(labels).not.toContain('and');
    expect(labels).not.toContain('or');
    expect(labels).not.toContain('true');
  });

  it('suggests vars under vars dot-notation', () => {
    const catalog = buildResponseExpressionCatalog({
      variables: {
        campus_code: {
          origin: 'detected',
          type: 'string',
          required: true,
          locations: new Set(['url_path']),
        },
        is_enabled: {
          origin: 'detected',
          type: 'boolean',
          required: false,
          locations: new Set(['body']),
        },
      },
    });

    const rootResult = getResponseExpressionCompletions({
      doc: '',
      cursor: 0,
      catalog,
      explicit: true,
    });
    expect(rootResult?.options.map((option) => option.label)).toContain('vars');

    const memberResult = getResponseExpressionCompletions({
      doc: 'vars.',
      cursor: 'vars.'.length,
      catalog,
      explicit: false,
    });

    expect(memberResult?.options.map((option) => option.label)).toEqual(
      expect.arrayContaining(['campus_code', 'is_enabled']),
    );
  });

  it('auto-triggers after logical operators and group openings', () => {
    expect(shouldAutoTriggerCompletion('', 'response.body.ok == ', 'response.body.ok == '.length)).toBe(true);
    expect(
      shouldAutoTriggerCompletion(
        'response.body.ok == true and',
        'response.body.ok == true and ',
        'response.body.ok == true and '.length,
      ),
    ).toBe(true);
    expect(shouldAutoTriggerCompletion('', '(', '('.length)).toBe(true);
  });
});

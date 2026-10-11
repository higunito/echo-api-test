-- 性能確認用（SC-003）: 募集中・受注済・完了・期限切れが混ざったクエストを 1,000 件投入する。
-- 冒険者の初期データ（seed.sql）を先に投入しておくこと。
INSERT INTO quests (id, title, description, reward, deadline, status,
                    assignee_id, accepted_at, completed_at, created_at, version)
SELECT gen_random_uuid(),
       '大量クエスト #' || i,
       '性能確認用のクエスト',
       (i * 10) % 5000,
       CASE WHEN i % 4 = 0 THEN now() - interval '1 day' ELSE now() + interval '7 days' END,
       CASE WHEN i % 5 = 1 THEN 'accepted' WHEN i % 5 = 2 THEN 'completed' ELSE 'open' END,
       CASE WHEN i % 5 IN (1, 2) THEN '0199d000-0000-7000-8000-000000000001'::uuid END,
       CASE WHEN i % 5 IN (1, 2) THEN now() - interval '1 hour' END,
       CASE WHEN i % 5 = 2 THEN now() - interval '30 minutes' END,
       now() - (i || ' minutes')::interval,
       1
FROM generate_series(1, 1000) AS i;

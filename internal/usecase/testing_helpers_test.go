package usecase

// testUserID は各Usecaseのテストで共通して使う、userIDスコープの検証用の固定値。
// 実際の値そのものに意味はなく、各Usecaseのrepo呼び出しに一貫して伝播していることを
// 確認するために使う。
const testUserID int64 = 42

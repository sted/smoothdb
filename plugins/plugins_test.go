package plugins

// type myHost struct{}

// func (*myHost) GetLogger() *logging.Logger {
// 	zlogger := zerolog.New(os.Stderr).With().Timestamp().Logger()
// 	return &logging.Logger{Logger: &zlogger}
// }

// func (*myHost) GetRouter() *heligo.Router {
// 	return heligo.New()
// }

// func (*myHost) GetDBE() *database.DbEngine {
// 	return nil
// }

// @@ enable it when we discover how to test plugins in vscode
// func TestPlugins(t *testing.T) {
// 	pm := InitPluginManager(&myHost{}, "../_plugins", []string{"example"})
// 	if pm == nil {
// 		t.Error()
// 	}
// 	err := pm.Load()
// 	if err != nil {
// 		t.Error()
// 	}
// }

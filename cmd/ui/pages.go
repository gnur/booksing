package main

/*
func (app *booksingApp) deleteBook(c *gin.Context) {
	hash := c.Param("hash")

	book, err := app.searchDB.GetBook(hash)
	if err != nil {
		c.HTML(404, "error.html", V{
			Error: errors.New("Book not found"),
		})
		return
	}

	err = os.Remove(book.Path)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
			"path": book.Path,
		}).Error("Could not delete book from filesystem")
		c.HTML(500, "error.html", V{
			Error: fmt.Errorf("Unable to delete book from filesystem: %w", err),
		})
		return
	}

	err = app.searchDB.DeleteBook(hash)
	if err != nil {
		app.logger.WithFields(logrus.Fields{
			"hash": hash,
			"err":  err,
		}).Error("Could not delete book from database")
		c.HTML(500, "error.html", V{
			Error: fmt.Errorf("Unable to delete book from database: %w", err),
		})
		return
	}
	app.logger.WithFields(logrus.Fields{
		"hash": hash,
	}).Info("book was deleted")
	c.Redirect(302, c.Request.Referer())
}
*/

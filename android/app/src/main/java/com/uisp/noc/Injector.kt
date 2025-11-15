package com.uisp.noc

import android.app.Application
import android.content.Context
import androidx.lifecycle.ViewModelProvider
import com.uisp.noc.data.SessionStore
import com.uisp.noc.data.UispRepository
import com.uisp.noc.ui.MainViewModel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.runBlocking

/**
 * A simple dependency injector to manage the creation of key components.
 */
object Injector {

    private var repository: UispRepository? = null

    /**
     * Creates and returns the [MainViewModel.Factory].
     *
     * This function ensures that the [UispRepository] is instantiated on a background
     * thread the first time it's needed.
     */
    fun provideViewModelFactory(context: Context): ViewModelProvider.Factory {
        val application = context.applicationContext as Application
        val sessionStore = SessionStore.getInstance(application)
        val repo = getRepository()
        return MainViewModel.Factory(application, repo, sessionStore)
    }

    /**
     * Returns the singleton instance of the [UispRepository], creating it on a
     * background thread if it doesn't exist yet.
     */
    fun getRepository(): UispRepository {
        return repository ?: synchronized(this) {
            repository ?: runBlocking(Dispatchers.IO) {
                UispRepository().also {
                    repository = it
                }
            }
        }
    }
}

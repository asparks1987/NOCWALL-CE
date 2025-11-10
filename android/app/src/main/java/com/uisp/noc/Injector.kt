package com.uisp.noc

import android.content.Context
import com.uisp.noc.data.SessionStore
import com.uisp.noc.data.UispRepository
import com.uisp.noc.ui.MainViewModel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

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
    suspend fun provideViewModelFactory(context: Context): MainViewModel.Factory {
        val sessionStore = SessionStore.getInstance(context.applicationContext)
        val repo = getRepository()
        return MainViewModel.Factory(repo, sessionStore)
    }

    /**
     * Returns the singleton instance of the [UispRepository], creating it on a
     * background thread if it doesn't exist yet.
     */
    private suspend fun getRepository(): UispRepository {
        return withContext(Dispatchers.IO) {
            repository ?: UispRepository().also {
                repository = it
            }
        }
    }
}

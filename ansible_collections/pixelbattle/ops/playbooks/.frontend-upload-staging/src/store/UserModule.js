const state = {
    isAuthorized: false,
    id: null
}

const getters = {
    getAuthorized: (state) => state.isAuthorized,
    getID: (state) => state.id,
}

const mutations = {
    setAuthorized(state, authorized) {
        state.isAuthorized = authorized
    },
    setID(state, id) {
        state.id = id
    }
}

export default {
    namespaced: true,
    state, 
    getters,
    mutations
}
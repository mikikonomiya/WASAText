import axios from "axios";

const instance = axios.create({
	baseURL: __API_URL__,
	timeout: 1000 * 15
});

instance.interceptors.request.use(
	(config) => {
		const token = localStorage.getItem("token"); // checks for token in localStorage
		if (token) {
			config.headers["Authorization"] = `Bearer ${token}`; // adds token to the request header
		}
		return config;
	},
	// to handle errors happening while trying to send a request
	(error) => {
		return Promise.reject(error);
	}
);

export default instance;

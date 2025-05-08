import api from '../services/api';

export const getCurrentUser = async () => {
  return await api.get('/auth/user/info');
};
